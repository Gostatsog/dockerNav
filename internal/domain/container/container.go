package container

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/internal/ui"
)

// Service provides container operations
type Service struct {
	dockerClient *client.DockerClient
}

// NewService creates a new container service
func NewService(dockerClient *client.DockerClient) *Service {
	return &Service{
		dockerClient: dockerClient,
	}
}

// ListContainers displays a list of containers
func (s *Service) ListContainers(all bool) {
	options := container.ListOptions{
		All: all,
	}

	containers, err := s.dockerClient.Client.ContainerList(s.dockerClient.Ctx, options)
	if err != nil {
		fmt.Printf("Error listing containers: %v\n", err)
		ui.WaitForEnter()
		return
	}

	if len(containers) == 0 {
		fmt.Println("No containers found")
		ui.WaitForEnter()
		return
	}

	// Prepare table data
	rows := make([][]string, 0, len(containers))
	for _, container := range containers {
		// Get a nice name (remove the leading slash)
		name := strings.TrimPrefix(container.Names[0], "/")
		
		// Format ports
		ports := s.formatPorts(container.Ports)
		
		rows = append(rows, []string{
			name,
			container.ID[:12],
			container.Image,
			container.Status,
			ports,
		})
	}

	// Display table
	tableOptions := ui.TableOptions{
		Headers: []string{"NAME", "ID", "IMAGE", "STATUS", "PORTS"},
		Rows:    rows,
		HeaderColors: ui.DefaultHeaderColors(5),
	}

	ui.RenderTable(tableOptions)
	ui.WaitForEnter()
}

// formatPorts creates a readable string from port mappings
func (s *Service) formatPorts(ports []types.Port) string {
	if len(ports) == 0 {
		return ""
	}

	var portStrings []string
	for _, port := range ports {
		if port.PublicPort != 0 {
			portStr := fmt.Sprintf("%d:%d/%s", port.PublicPort, port.PrivatePort, port.Type)
			portStrings = append(portStrings, portStr)
		} else {
			portStr := fmt.Sprintf("%d/%s", port.PrivatePort, port.Type)
			portStrings = append(portStrings, portStr)
		}
	}

	return strings.Join(portStrings, ", ")
}

// SelectContainer allows the user to select a container
func (s *Service) SelectContainer() (container.Summary, error) {
	// Get all containers
	options := container.ListOptions{
		All: true,
	}

	containers, err := s.dockerClient.Client.ContainerList(s.dockerClient.Ctx, options)
	if err != nil {
		return container.Summary{}, err
	}

	if len(containers) == 0 {
		fmt.Println("No containers found")
		ui.WaitForEnter()
		return container.Summary{}, errors.New("no containers found")
	}

	// Create container name list for selection
	var containerOptions []string
	containerMap := make(map[string]container.Summary)

	for _, container := range containers {
		name := strings.TrimPrefix(container.Names[0], "/")
		status := container.Status
		option := fmt.Sprintf("%-20s [%s] [%s]", name, container.ID[:12], status)
		containerOptions = append(containerOptions, option)
		containerMap[option] = container
	}
	
	containerOptions = append(containerOptions, "Back")

	// Prompt for container selection
	selectOptions := ui.SelectOptions{
		Label: "Select Container",
		Items: containerOptions,
		Size:  10,
	}

	_, result, err := ui.Select(selectOptions)
	if err != nil {
		return container.Summary{}, err
	}

	if result == "Back" {
		return container.Summary{}, errors.New("back selected")
	}

	// Return selected container
	return containerMap[result], nil
}

// DisplayContainerActions shows available actions for a container
func (s *Service) DisplayContainerActions(containerData container.Summary) {
	containerName := strings.TrimPrefix(containerData.Names[0], "/")
	title := fmt.Sprintf("Actions for container: %s", containerName)
	
	// Different actions based on container state
	isRunning := strings.Contains(containerData.Status, "Up")
	
	var actions []string
	if isRunning {
		actions = []string{
			"Logs",
			"Exec (Run command)",
			"Stop",
			"Restart",
			"Inspect",
			"Stats",
			"Port Mappings",
			"Back",
		}
	} else {
		actions = []string{
			"Start",
			"Remove",
			"Inspect",
			"Logs (previous run)",
			"Port Mappings",
			"Back",
		}
	}

	options := ui.SelectOptions{
		Label: title,
		Items: actions,
	}

	_, result, err := ui.Select(options)
	if err != nil {
		fmt.Printf("Error selecting action: %v\n", err)
		return
	}

	switch result {
	case "Logs":
		s.showContainerLogs(containerData)
	case "Logs (previous run)":
		s.showContainerLogs(containerData)
	case "Exec (Run command)":
		s.execInContainer(containerData)
	case "Stop":
		s.stopContainer(containerData)
	case "Start":
		s.startContainer(containerData)
	case "Restart":
		s.restartContainer(containerData)
	case "Remove":
		s.removeContainer(containerData)
	case "Inspect":
		s.inspectContainer(containerData)
	case "Stats":
		s.showContainerStats(containerData)
	case "Port Mappings":
		s.showContainerPorts(containerData)
	case "Back":
		return
	}
}

// showContainerLogs displays logs for a container
func (s *Service) showContainerLogs(containerData container.Summary) {
	// Implementation omitted for brevity - would follow the same pattern as in the original code
	fmt.Println("Showing logs for container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// execInContainer executes a command in a running container
func (s *Service) execInContainer(containerData container.Summary) {
	// Implementation omitted for brevity
	fmt.Println("Executing command in container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// stopContainer stops a running container
func (s *Service) stopContainer(containerData container.Summary) {
	containerName := strings.TrimPrefix(containerData.Names[0], "/")
	
	// Confirm action
	confirmed, err := ui.Confirm(fmt.Sprintf("Stop container %s?", containerName))
	if err != nil {
		fmt.Printf("Error confirming action: %v\n", err)
		return
	}

	if !confirmed {
		return
	}

	// Stop container with a 10 second timeout
	timeout := 10 * time.Second
	timeoutSeconds := int(timeout.Seconds())
	err = s.dockerClient.Client.ContainerStop(s.dockerClient.Ctx, containerData.ID, container.StopOptions{Timeout: &timeoutSeconds})
	if err != nil {
		fmt.Printf("Error stopping container: %v\n", err)
		ui.WaitForEnter()
		return
	}

	fmt.Printf("Container %s stopped successfully\n", containerName)
	ui.WaitForEnter()
}

// startContainer starts a stopped container
func (s *Service) startContainer(containerData container.Summary) {
	// Implementation omitted for brevity
	fmt.Println("Starting container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// restartContainer restarts a container
func (s *Service) restartContainer(containerData container.Summary) {
	// Implementation omitted for brevity
	fmt.Println("Restarting container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// removeContainer removes a container
func (s *Service) removeContainer(containerData container.Summary) {
	// Implementation omitted for brevity
	fmt.Println("Removing container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// inspectContainer shows detailed information about a container
func (s *Service) inspectContainer(containerData container.Summary) {
	// Implementation omitted for brevity
	fmt.Println("Inspecting container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// showContainerStats shows live stats for a container
func (s *Service) showContainerStats(containerData container.Summary) {
	// Implementation omitted for brevity
	fmt.Println("Showing stats for container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// showContainerPorts displays port mappings for a container
func (s *Service) showContainerPorts(containerData container.Summary) {
	// Implementation omitted for brevity
	fmt.Println("Showing port mappings for container:", strings.TrimPrefix(containerData.Names[0], "/"))
	ui.WaitForEnter()
}

// RunNewContainer guides the user through creating a new container
func (s *Service) RunNewContainer() error {
	// Implementation omitted for brevity - would follow the same pattern as in the original code
	fmt.Println("Run container wizard...")
	ui.WaitForEnter()
	return nil
}