package menu

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/internal/ui"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// ContainerMenu represents the container menu
type ContainerMenu struct {
	ctx          context.Context
	dockerClient *client.DockerClient
	display      *ui.Display
}

// NewContainerMenu creates a new instance of the container menu
func NewContainerMenu(ctx context.Context, dockerClient *client.DockerClient) *ContainerMenu {
	return &ContainerMenu{
		ctx:          ctx,
		dockerClient: dockerClient,
		display:      ui.NewDisplay(),
	}
}

// Display shows the container menu and handles user input
func (m *ContainerMenu) Display() {
	for {
		// Clear the screen
		m.display.ClearScreen()

		// Display containers
		m.listContainers()

		// Display menu options
		m.display.PrintTitle("Container Management")
		menuItems := []string{
			"1. List Containers",
			"2. Start Container",
			"3. Stop Container",
			"4. Restart Container",
			"5. Remove Container",
			"6. View Container Logs",
			"7. Run New Container",
			"0. Back to Main Menu",
		}
		m.display.PrintMenu(menuItems)

		// Get user choice
		choice := ui.GetUserInput("Enter your choice: ")

		// Process the user's choice
		switch choice {
		case "1":
			m.listContainers()
			ui.PressEnterToContinue()
		case "2":
			m.startContainer()
		case "3":
			m.stopContainer()
		case "4":
			m.restartContainer()
		case "5":
			m.removeContainer()
		case "6":
			m.viewContainerLogs()
		case "7":
			m.runNewContainer()
		case "0", "back", "b":
			return
		default:
			m.display.PrintError("Invalid choice. Please try again.")
			ui.PressEnterToContinue()
		}
	}
}

// listContainers lists all containers
func (m *ContainerMenu) listContainers() {
	m.display.PrintTitle("Containers")

	// Get containers
	containers, err := m.dockerClient.Client.ContainerList(m.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching containers: %v", err))
		return
	}

	if len(containers) == 0 {
		m.display.PrintMessage("No containers found.")
		return
	}

	// Prepare data for table
	headers := []string{"ID", "Name", "Image", "Status", "Created", "Ports"}
	data := [][]string{}

	for _, c := range containers {
		// Get container name without leading slash
		name := strings.TrimPrefix(c.Names[0], "/")

		// Format ports
		ports := []string{}
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			}
		}
		portsStr := strings.Join(ports, ", ")
		if portsStr == "" {
			portsStr = "None"
		}

		// Format created time (for simplicity, just showing seconds ago)
		createdTime := fmt.Sprintf("%d seconds ago", uint64(m.dockerClient.Client.DaemonHost()[0]-c.Created))

		// Add row to data
		data = append(data, []string{
			c.ID[:12],
			name,
			c.Image,
			c.Status,
			createdTime,
			portsStr,
		})
	}

	// Display table
	m.display.PrintTable(headers, data)
}

// startContainer starts a stopped container
func (m *ContainerMenu) startContainer() {
	// Get stopped containers
	filterArgs := filters.NewArgs()
	filterArgs.Add("status", "exited")
	containers, err := m.dockerClient.Client.ContainerList(m.ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching containers: %v", err))
		ui.PressEnterToContinue()
		return
	}

	if len(containers) == 0 {
		m.display.PrintMessage("No stopped containers to start.")
		ui.PressEnterToContinue()
		return
	}

	// List stopped containers
	m.display.PrintTitle("Stopped Containers")
	options := []string{}
	for i, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		options = append(options, fmt.Sprintf("%d. %s (%s)", i+1, name, c.ID[:12]))
	}
	m.display.PrintMenu(options)

	// Get user choice
	choice := ui.GetUserInput("Enter container number to start (or 0 to cancel): ")
	if choice == "0" {
		return
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(containers) {
		m.display.PrintError("Invalid choice. Please try again.")
		ui.PressEnterToContinue()
		return
	}

	// Start the container
	containerId := containers[idx-1].ID
	err = m.dockerClient.Client.ContainerStart(m.ctx, containerId, types.ContainerStartOptions{})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error starting container: %v", err))
	} else {
		name := strings.TrimPrefix(containers[idx-1].Names[0], "/")
		m.display.PrintSuccess(fmt.Sprintf("Container '%s' started successfully.", name))
	}
	ui.PressEnterToContinue()
}

// stopContainer stops a running container
func (m *ContainerMenu) stopContainer() {
	// Get running containers
	filterArgs := filters.NewArgs()
	filterArgs.Add("status", "running")
	containers, err := m.dockerClient.Client.ContainerList(m.ctx, types.ContainerListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching containers: %v", err))
		ui.PressEnterToContinue()
		return
	}

	if len(containers) == 0 {
		m.display.PrintMessage("No running containers to stop.")
		ui.PressEnterToContinue()
		return
	}

	// List running containers
	m.display.PrintTitle("Running Containers")
	options := []string{}
	for i, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		options = append(options, fmt.Sprintf("%d. %s (%s)", i+1, name, c.ID[:12]))
	}
	m.display.PrintMenu(options)

	// Get user choice
	choice := ui.GetUserInput("Enter container number to stop (or 0 to cancel): ")
	if choice == "0" {
		return
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(containers) {
		m.display.PrintError("Invalid choice. Please try again.")
		ui.PressEnterToContinue()
		return
	}

	// Stop the container
	containerId := containers[idx-1].ID
	timeout := 10 // seconds
	err = m.dockerClient.Client.ContainerStop(m.ctx, containerId, container.StopOptions{Timeout: &timeout})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error stopping container: %v", err))
	} else {
		name := strings.TrimPrefix(containers[idx-1].Names[0], "/")
		m.display.PrintSuccess(fmt.Sprintf("Container '%s' stopped successfully.", name))
	}
	ui.PressEnterToContinue()
}

// restartContainer restarts a container
func (m *ContainerMenu) restartContainer() {
	// Get all containers
	containers, err := m.dockerClient.Client.ContainerList(m.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching containers: %v", err))
		ui.PressEnterToContinue()
		return
	}

	if len(containers) == 0 {
		m.display.PrintMessage("No containers to restart.")
		ui.PressEnterToContinue()
		return
	}

	// List containers
	m.display.PrintTitle("All Containers")
	options := []string{}
	for i, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		options = append(options, fmt.Sprintf("%d. %s (%s) - %s", i+1, name, c.ID[:12], c.Status))
	}
	m.display.PrintMenu(options)

	// Get user choice
	choice := ui.GetUserInput("Enter container number to restart (or 0 to cancel): ")
	if choice == "0" {
		return
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(containers) {
		m.display.PrintError("Invalid choice. Please try again.")
		ui.PressEnterToContinue()
		return
	}

	// Restart the container
	containerId := containers[idx-1].ID
	timeout := 10 // seconds
	err = m.dockerClient.Client.ContainerRestart(m.ctx, containerId, container.StopOptions{Timeout: &timeout})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error restarting container: %v", err))
	} else {
		name := strings.TrimPrefix(containers[idx-1].Names[0], "/")
		m.display.PrintSuccess(fmt.Sprintf("Container '%s' restarted successfully.", name))
	}
	ui.PressEnterToContinue()
}

// removeContainer removes a container
func (m *ContainerMenu) removeContainer() {
	// Get all containers
	containers, err := m.dockerClient.Client.ContainerList(m.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching containers: %v", err))
		ui.PressEnterToContinue()
		return
	}

	if len(containers) == 0 {
		m.display.PrintMessage("No containers to remove.")
		ui.PressEnterToContinue()
		return
	}

	// List containers
	m.display.PrintTitle("All Containers")
	options := []string{}
	for i, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		options = append(options, fmt.Sprintf("%d. %s (%s) - %s", i+1, name, c.ID[:12], c.Status))
	}
	m.display.PrintMenu(options)

	// Get user choice
	choice := ui.GetUserInput("Enter container number to remove (or 0 to cancel): ")
	if choice == "0" {
		return
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(containers) {
		m.display.PrintError("Invalid choice. Please try again.")
		ui.PressEnterToContinue()
		return
	}

	// Ask for force option
	forceStr := ui.GetUserInput("Force remove running container? (y/N): ")
	force := strings.ToLower(forceStr) == "y" || strings.ToLower(forceStr) == "yes"

	// Remove the container
	containerId := containers[idx-1].ID
	err = m.dockerClient.Client.ContainerRemove(m.ctx, containerId, types.ContainerRemoveOptions{
		Force: force,
	})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error removing container: %v", err))
	} else {
		name := strings.TrimPrefix(containers[idx-1].Names[0], "/")
		m.display.PrintSuccess(fmt.Sprintf("Container '%s' removed successfully.", name))
	}
	ui.PressEnterToContinue()
}

// viewContainerLogs shows the logs of a container
func (m *ContainerMenu) viewContainerLogs() {
	// Get all containers
	containers, err := m.dockerClient.Client.ContainerList(m.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching containers: %v", err))
		ui.PressEnterToContinue()
		return
	}

	if len(containers) == 0 {
		m.display.PrintMessage("No containers available.")
		ui.PressEnterToContinue()
		return
	}

	// List containers
	m.display.PrintTitle("All Containers")
	options := []string{}
	for i, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		options = append(options, fmt.Sprintf("%d. %s (%s) - %s", i+1, name, c.ID[:12], c.Status))
	}
	m.display.PrintMenu(options)

	// Get user choice
	choice := ui.GetUserInput("Enter container number to view logs (or 0 to cancel): ")
	if choice == "0" {
		return
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(containers) {
		m.display.PrintError("Invalid choice. Please try again.")
		ui.PressEnterToContinue()
		return
	}

	// Get logs options
	tailStr := ui.GetUserInput("Number of lines to show (default: 100): ")
	tail := "100"
	if tailStr != "" {
		tail = tailStr
	}

	// Get container logs
	containerId := containers[idx-1].ID
	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	}
	
	logs, err := m.dockerClient.Client.ContainerLogs(m.ctx, containerId, logOptions)
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error getting container logs: %v", err))
		ui.PressEnterToContinue()
		return
	}
	defer logs.Close()

	// Display logs
	m.display.ClearScreen()
	name := strings.TrimPrefix(containers[idx-1].Names[0], "/")
	m.display.PrintTitle(fmt.Sprintf("Logs for '%s'", name))
	
	// Read logs - this is simplified, in a real implementation you'd want to handle
	// the byte stream properly accounting for Docker log format
	buf := make([]byte, 4096)
	for {
		_, err := logs.Read(buf)
		if err != nil {
			break
		}
		// Skip the first 8 bytes of each line (Docker log header)
		fmt.Print(string(buf[8:]))
	}
	
	ui.PressEnterToContinue()
}

// runNewContainer runs a new container
func (m *ContainerMenu) runNewContainer() {
	// Get available images
	images, err := m.dockerClient.Client.ImageList(m.ctx, types.ImageListOptions{})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching images: %v", err))
		ui.PressEnterToContinue()
		return
	}

	if len(images) == 0 {
		m.display.PrintMessage("No images available. Please pull an image first.")
		ui.PressEnterToContinue()
		return
	}

	// List images
	m.display.PrintTitle("Available Images")
	options := []string{}
	for i, img := range images {
		repoTags := "<none>"
		if len(img.RepoTags) > 0 {
			repoTags = img.RepoTags[0]
		}
		options = append(options, fmt.Sprintf("%d. %s", i+1, repoTags))
	}
	m.display.PrintMenu(options)

	// Get user choice
	choice := ui.GetUserInput("Enter image number to use (or 0 to cancel): ")
	if choice == "0" {
		return
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(images) {
		m.display.PrintError("Invalid choice. Please try again.")
		ui.PressEnterToContinue()
		return
	}

	// Get container name
	name := ui.GetUserInput("Enter container name (leave empty for random name): ")

	// Get container command
	cmd := ui.GetUserInput("Enter container command (leave empty for default): ")
	
	// Get port mapping
	portMapping := ui.GetUserInput("Enter port mapping (e.g., 8080:80) or leave empty: ")
	
	// Create container configuration
	config := &container.Config{
		Image: images[idx-1].RepoTags[0],
	}
	
	if cmd != "" {
		config.Cmd = strings.Split(cmd, " ")
	}
	
	hostConfig := &container.HostConfig{}
	
	if portMapping != "" {
		parts := strings.Split(portMapping, ":")
		if len(parts) == 2 {
			hostPort := parts[0]
			containerPort := parts[1]
			hostConfig.PortBindings = map[string][]string{
				containerPort + "/tcp": {{HostPort: hostPort}},
			}
			config.ExposedPorts = map[string]struct{}{
				containerPort + "/tcp": {},
			}
		}
	}
	
	// Create container
	resp, err := m.dockerClient.Client.ContainerCreate(
		m.ctx,
		config,
		hostConfig,
		nil,
		nil,
		name,
	)
	
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error creating container: %v", err))
		ui.PressEnterToContinue()
		return
	}
	
	// Start the container
	err = m.dockerClient.Client.ContainerStart(m.ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error starting container: %v", err))
	} else {
		m.display.PrintSuccess(fmt.Sprintf("Container created and started with ID: %s", resp.ID[:12]))
	}
	ui.PressEnterToContinue()
}