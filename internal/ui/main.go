package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/Gostatsog/dockerNav/internal/client"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

// DockerInfoMsg contains Docker daemon information
type DockerInfoMsg struct {
	ContainerCount int
	ImageCount     int
	ServerVersion  string
	EngineVersion  string
	Error          error
}

// View represents the different screens in the application
type View int

const (
	ViewMain View = iota
	ViewContainers
	ViewImages
	ViewNetworks
	ViewVolumes
	ViewSystem
)

// MainModel is the root model for the application
type MainModel struct {
	dockerClient   *client.DockerClient
	currentView    View
	containers     *ContainerModel
	images         *ImageModel
	networks       *NetworkModel
	volumes        *VolumeModel
	system         *SystemModel
	width          int
	height         int
	containerCount int
	imageCount     int
	serverVersion  string
	engineVersion  string
	loading        bool
	error          error
}

// NewMainModel creates and initializes the main model
func NewMainModel() tea.Model {
	// Create Docker client
	dockerClient, err := client.NewDockerClient(context.Background())
	if err != nil {
		// If we can't connect to Docker, we'll show an error after initialization
		return &MainModel{
			error:   err,
			loading: false,
		}
	}

	m := &MainModel{
		dockerClient: dockerClient,
		currentView:  ViewMain,
		loading:      true,
	}

	// Initialize sub-models for each view
	m.containers = NewContainerModel(dockerClient)
	m.images = NewImageModel(dockerClient)
	m.networks = NewNetworkModel(dockerClient)
	m.volumes = NewVolumeModel(dockerClient)
	m.system = NewSystemModel(dockerClient)

	return m
}

// Init implements tea.Model and returns the initial command
func (m *MainModel) Init() tea.Cmd {
	// Start by fetching Docker information
	return m.fetchDockerInfo()
}

// fetchDockerInfo returns a command that retrieves Docker information
func (m *MainModel) fetchDockerInfo() tea.Cmd {
	return func() tea.Msg {
		if m.dockerClient == nil {
			return DockerInfoMsg{
				Error: fmt.Errorf("docker client not initialized"),
			}
		}

		ctx := m.dockerClient.Ctx

		// Get container count
		containers, err := m.dockerClient.Client.ContainerList(ctx, container.ListOptions{All: true})
		if err != nil {
			return DockerInfoMsg{Error: err}
		}

		// Get image count
		images, err := m.dockerClient.Client.ImageList(ctx, image.ListOptions{})
		if err != nil {
			return DockerInfoMsg{Error: err}
		}

		// Get Docker version
		version, err := m.dockerClient.Client.ServerVersion(ctx)
		if err != nil {
			return DockerInfoMsg{Error: err}
		}

		return DockerInfoMsg{
			ContainerCount: len(containers),
			ImageCount:     len(images),
			ServerVersion:  version.Version,
			EngineVersion:  version.Version,
			Error:          nil,
		}
	}
}

// Update handles messages and updates the model
func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "1":
			m.currentView = ViewContainers
			return m, m.containers.Init()

		case "2":
			m.currentView = ViewImages
			return m, m.images.Init()

		case "3":
			m.currentView = ViewNetworks
			return m, m.networks.Init()

		case "4":
			m.currentView = ViewVolumes
			return m, m.volumes.Init()

		case "5":
			m.currentView = ViewSystem
			return m, m.system.Init()

		case "0", "esc":
			if m.currentView != ViewMain {
				m.currentView = ViewMain
				return m, m.fetchDockerInfo()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update sub-model dimensions too
		if m.containers != nil {
			m.containers.width = msg.Width
			m.containers.height = msg.Height
		}
		if m.images != nil {
			m.images.width = msg.Width
			m.images.height = msg.Height
		}
		if m.networks != nil {
			m.networks.width = msg.Width
			m.networks.height = msg.Height
		}
		if m.volumes != nil {
			m.volumes.width = msg.Width
			m.volumes.height = msg.Height
		}
		if m.system != nil {
			m.system.width = msg.Width
			m.system.height = msg.Height
		}

	case DockerInfoMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		m.containerCount = msg.ContainerCount
		m.imageCount = msg.ImageCount
		m.serverVersion = msg.ServerVersion
		m.engineVersion = msg.EngineVersion
	}

	// Update the current view
	switch m.currentView {
	case ViewContainers:
		var newModel tea.Model
		newModel, cmd = m.containers.Update(msg)
		if newContainerModel, ok := newModel.(*ContainerModel); ok {
			m.containers = newContainerModel
		}
		cmds = append(cmds, cmd)

	case ViewImages:
		var newModel tea.Model
		newModel, cmd = m.images.Update(msg)
		if newImageModel, ok := newModel.(*ImageModel); ok {
			m.images = newImageModel
		}
		cmds = append(cmds, cmd)

	case ViewNetworks:
		var newModel tea.Model
		newModel, cmd = m.networks.Update(msg)
		if newNetworkModel, ok := newModel.(*NetworkModel); ok {
			m.networks = newNetworkModel
		}
		cmds = append(cmds, cmd)

	case ViewVolumes:
		var newModel tea.Model
		newModel, cmd = m.volumes.Update(msg)
		if newVolumeModel, ok := newModel.(*VolumeModel); ok {
			m.volumes = newVolumeModel
		}
		cmds = append(cmds, cmd)

	case ViewSystem:
		var newModel tea.Model
		newModel, cmd = m.system.Update(msg)
		if newSystemModel, ok := newModel.(*SystemModel); ok {
			m.system = newSystemModel
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m *MainModel) View() string {
	if m.error != nil {
		errorBox := StyleInfoBox.
			BorderForeground(ColorError).
			Render(StyleError.Render("Error connecting to Docker: " + m.error.Error()))
		
		helpText := "\nMake sure Docker is running and try again.\nPress q to quit."
		
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleTitle.Render("DockerNav Error"),
				errorBox,
				helpText,
			),
		)
	}

	if m.loading {
		return StyleMainLayout.Render("Loading Docker information...")
	}

	switch m.currentView {
	case ViewMain:
		return m.renderMainView()
	case ViewContainers:
		return m.containers.View()
	case ViewImages:
		return m.images.View()
	case ViewNetworks:
		return m.networks.View()
	case ViewVolumes:
		return m.volumes.View()
	case ViewSystem:
		return m.system.View()
	default:
		return "Invalid view"
	}
}

// renderMainView renders the main menu
func (m *MainModel) renderMainView() string {
	title := StyleTitle.Render("DockerNav - Main Menu")

	// Create info box with Docker information
	infoContent := fmt.Sprintf(
		"Docker Version: %s\nAPI Version: %s\nContainers: %d\nImages: %d",
		m.serverVersion,
		m.engineVersion,
		m.containerCount,
		m.imageCount,
	)
	infoBox := StyleInfoBox.Render(infoContent)

	// Create menu options
	menuItems := []string{
		"1. Container Management",
		"2. Image Management",
		"3. Network Management",
		"4. Volume Management",
		"5. System Management",
		"0. Exit",
	}
	menu := StyleMenu.Render(strings.Join(menuItems, "\n"))

	// Create footer with help text
	footer := StyleFooter.Render("Press q to quit. Use numbers to navigate.")

	// Join everything together
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		infoBox,
		"",
		menu,
		"",
		footer,
	)

	return StyleMainLayout.Render(content)
}