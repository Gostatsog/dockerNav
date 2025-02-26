package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/Gostatsog/dockerNav/internal/client"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// ContainerCreateMsg carries the result of container creation
type ContainerCreateMsg struct {
	ContainerID string
	Error       error
}

// ContainerCreateModel manages the container creation form
type ContainerCreateModel struct {
	docker     *client.DockerClient
	form       *huh.Form
	networks   []network.Summary
	images     []string
	width      int
	height     int
	error      error
	result     string
	
	// Form values
	imageName   string
	containerName string
	ports       string
	volumes     string
	envVars     string
	command     string
	networkName string
	restart     string
}

// NewContainerCreateModel creates a new container creation model
func NewContainerCreateModel(docker *client.DockerClient) *ContainerCreateModel {
	m := &ContainerCreateModel{
		docker:     docker,
		restart:    "no", // Default restart policy
	}
	
	return m
}

// Init initializes the model
func (m *ContainerCreateModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchNetworks(),
		m.fetchImages(),
	)
}

// fetchNetworks retrieves available networks
func (m *ContainerCreateModel) fetchNetworks() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		networks, err := m.docker.Client.NetworkList(ctx, network.ListOptions{})
		if err != nil {
			return NetworkListMsg{
				Networks: []network.Summary{},
				Error:    err,
			}
		}
		
		// Store networks for the form
		m.networks = networks
		
		// Initialize the form after fetching data
		m.initForm()
		
		return NetworkListMsg{
			Networks: networks,
			Error:    nil,
		}
	}
}

// fetchImages retrieves available images
func (m *ContainerCreateModel) fetchImages() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		images, err := m.docker.Client.ImageList(ctx, image.ListOptions{})
		if err != nil {
			return ImageListMsg{
				Images: []image.Summary{},
				Error:  err,
			}
		}
		
		// Extract image names and tags
		imageNames := make([]string, 0, len(images))
		for _, img := range images {
			if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
				imageNames = append(imageNames, img.RepoTags[0])
			}
		}
		
		m.images = imageNames
		
		return ImageListMsg{
			Images: images,
			Error:  nil,
		}
	}
}

// initForm initializes the form with the fetched data
func (m *ContainerCreateModel) initForm() {
	// Create network options
	networkOptions := []huh.Option[string]{}
	for _, net := range m.networks {
		networkOptions = append(networkOptions, huh.NewOption(net.Name, net.Name))
	}
	
	// Set default network to bridge if available
	defaultNetwork := "bridge"
	m.networkName = defaultNetwork
	
	// Create restart policy options
	restartOptions := []huh.Option[string]{
		huh.NewOption("No restart", "no"),
		huh.NewOption("Always restart", "always"),
		huh.NewOption("Restart on failure", "on-failure"),
		huh.NewOption("Restart unless stopped", "unless-stopped"),
	}
	
	// Create the form
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Image").
				Options(huh.NewOptions(m.images...)...).
				Value(&m.imageName),
			
			huh.NewInput().
				Title("Container Name").
				Placeholder("my-container").
				Value(&m.containerName),
			
			huh.NewInput().
				Title("Ports").
				Placeholder("host:container, e.g., 8080:80").
				Value(&m.ports),
			
			huh.NewInput().
				Title("Volumes").
				Placeholder("host:container, e.g., ./data:/data").
				Value(&m.volumes),
			
			huh.NewInput().
				Title("Environment Variables").
				Placeholder("KEY=VALUE, KEY2=VALUE2").
				Value(&m.envVars),
			
			huh.NewInput().
				Title("Command").
				Placeholder("Command to run, e.g., nginx -g 'daemon off;'").
				Value(&m.command),
			
			huh.NewSelect[string]().
				Title("Network").
				Options(networkOptions...).
				Value(&m.networkName),
			
			huh.NewSelect[string]().
				Title("Restart Policy").
				Options(restartOptions...).
				Value(&m.restart),
		),
	).WithWidth(m.width - 4).WithShowHelp(true)
}

// createContainer creates a Docker container based on form input
func (m *ContainerCreateModel) createContainer() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Parse ports
		portBindings := nat.PortMap{}
		exposedPorts := nat.PortSet{}
		
		if m.ports != "" {
			for _, binding := range strings.Split(m.ports, ",") {
				binding = strings.TrimSpace(binding)
				parts := strings.Split(binding, ":")
				
				if len(parts) != 2 {
					return ContainerCreateMsg{
						Error: fmt.Errorf("invalid port format: %s, expected host:container", binding),
					}
				}
				
				hostPort := parts[0]
				containerPort := parts[1]
				
				// Append protocol if not specified
				if !strings.Contains(containerPort, "/") {
					containerPort = containerPort + "/tcp"
				}
				
				natPort, err := nat.NewPort(strings.Split(containerPort, "/")[1], strings.Split(containerPort, "/")[0])
				if err != nil {
					return ContainerCreateMsg{
						Error: fmt.Errorf("invalid port: %s", err),
					}
				}
				
				portBindings[natPort] = []nat.PortBinding{
					{
						HostPort: hostPort,
					},
				}
				
				exposedPorts[natPort] = struct{}{}
			}
		}
		
		// Parse volumes
		volumes := []string{}
		if m.volumes != "" {
			volumes = strings.Split(m.volumes, ",")
			for i, v := range volumes {
				volumes[i] = strings.TrimSpace(v)
			}
		}
		
		// Parse environment variables
		env := []string{}
		if m.envVars != "" {
			env = strings.Split(m.envVars, ",")
			for i, e := range env {
				env[i] = strings.TrimSpace(e)
			}
		}
		
		// Parse command
		var cmd []string
		if m.command != "" {
			cmd = strings.Fields(m.command)
		}
		
		// Restart policy
		restartPolicy := container.RestartPolicy{
			Name: container.RestartPolicyMode(m.restart),
		}
		
		// Create container config
		config := &container.Config{
			Image:        m.imageName,
			ExposedPorts: exposedPorts,
			Env:          env,
			Cmd:          cmd,
		}
		
		// Create host config
		hostConfig := &container.HostConfig{
			PortBindings: portBindings,
			Binds:        volumes,
			RestartPolicy: restartPolicy,
		}
		
		// Create network config
		networkConfig := &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				m.networkName: {},
			},
		}
		
		// Create the container
		resp, err := m.docker.Client.ContainerCreate(
			ctx,
			config,
			hostConfig,
			networkConfig,
			nil,
			m.containerName,
		)
		
		if err != nil {
			return ContainerCreateMsg{
				Error: err,
			}
		}
		
		return ContainerCreateMsg{
			ContainerID: resp.ID,
			Error:       nil,
		}
	}
}

// Update handles messages and updates the model
func (m *ContainerCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update form width if it's initialized
		if m.form != nil {
			m.form = m.form.WithWidth(msg.Width - 4)
		}

	case NetworkListMsg:
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		m.networks = msg.Networks

	case ImageListMsg:
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}

	case ContainerCreateMsg:
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}

		m.result = fmt.Sprintf("Container created successfully with ID: %s", msg.ContainerID[:12])

	// Check if form is completed instead of huh.FormSubmitMsg
	default:
		if m.form != nil && m.form.State == huh.StateCompleted {
			return m, m.createContainer()
		}
	}

	// Correctly update the form with type assertion
	if m.form != nil {
		var cmd tea.Cmd
		var newForm tea.Model
		newForm, cmd = m.form.Update(msg)

		// Assert the returned model back to *huh.Form
		if updatedForm, ok := newForm.(*huh.Form); ok {
			m.form = updatedForm
		}

		return m, cmd
	}

	return m, nil
}
// View renders the current view
func (m *ContainerCreateModel) View() string {
	if m.error != nil {
		errorBox := StyleInfoBox.
			BorderForeground(ColorError).
			Render(StyleError.Render(fmt.Sprintf("Error: %v", m.error)))
		
		help := "Press esc to go back"
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render("Create Container"),
				errorBox, 
				help,
			),
		)
	}
	
	if m.result != "" {
		successBox := StyleInfoBox.
			BorderForeground(ColorSuccess).
			Render(StyleSuccess.Render(m.result))
		
		help := "Press esc to go back"
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render("Create Container"),
				successBox, 
				help,
			),
		)
	}
	
	if m.form == nil {
		return StyleMainLayout.Render("Loading...")
	}
	
	title := StyleTitle.Render("Create Container")
	formView := m.form.View()
	
	help := StyleHelp.Render(
		"↑/↓: Navigate • Tab: Next Field • Enter: Submit • Esc: Back",
	)
	
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		formView,
		"",
		help,
	)
	
	return StyleMainLayout.Render(content)
}