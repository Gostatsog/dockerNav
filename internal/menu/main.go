package menu

import (
	"context"
	"fmt"
	"os"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/internal/ui"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

// MainMenu represents the main menu of the application
type MainMenu struct {
	ctx          context.Context
	dockerClient *client.DockerClient
	display      *ui.Display
}

// NewMainMenu creates a new instance of the main menu
func NewMainMenu(ctx context.Context, dockerClient *client.DockerClient) *MainMenu {
	return &MainMenu{
		ctx:          ctx,
		dockerClient: dockerClient,
		display:      ui.NewDisplay(),
	}
}

// Display shows the main menu and handles user input
func (m *MainMenu) Display() {
	for {
		// Clear the screen
		m.display.ClearScreen()

		// Display Docker info
		m.displayDockerInfo()

		// Display menu options
		m.display.PrintTitle("DockerNav - Main Menu")
		menuItems := []string{
			"1. Container Management",
			"2. Image Management",
			"3. Network Management",
			"4. Volume Management",
			"5. System Management",
			"0. Exit",
		}
		m.display.PrintMenu(menuItems)

		// Get user choice
		choice := ui.ReadInput("Enter your choice: ")

		// Process the user's choice
		switch choice {
		case "1":
			containerMenu := NewContainerMenu(m.ctx, m.dockerClient)
			containerMenu.Display()
		case "2":
			imageMenu := NewImageMenu(m.ctx, m.dockerClient)
			imageMenu.Display()
		case "3":
			networkMenu := NewNetworkMenu(m.ctx, m.dockerClient)
			networkMenu.Display()
		case "4":
			volumeMenu := NewVolumeMenu(m.ctx, m.dockerClient)
			volumeMenu.Display()
		case "5":
			systemMenu := NewSystemMenu(m.ctx, m.dockerClient)
			systemMenu.Display()
		case "0", "exit", "quit", "q":
			m.display.PrintMessage("Exiting DockerNav. Goodbye!")
			os.Exit(0)
		default:
			m.display.PrintError("Invalid choice. Please try again.")
			ui.WaitForEnter()
		}
	}
}

// displayDockerInfo shows basic Docker information
func (m *MainMenu) displayDockerInfo() {
	m.display.PrintTitle("Docker Information")

	// Get Docker info
	info, err := m.dockerClient.Client.Info(m.ctx)
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching Docker info: %v", err))
		return
	}

	// Get Docker version
	version, err := m.dockerClient.Client.ServerVersion(m.ctx)
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching Docker version: %v", err))
		return
	}

	// Get containers count
	containers, err := m.dockerClient.Client.ContainerList(m.ctx, container.ListOptions{All: true})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching containers: %v", err))
		return
	}

	// Get images count
	images, err := m.dockerClient.Client.ImageList(m.ctx, image.ListOptions{All: true})
	if err != nil {
		m.display.PrintError(fmt.Sprintf("Error fetching images: %v", err))
		return
	}

	// Display information
	infoItems := []string{
		fmt.Sprintf("Docker Version: %s", version.Version),
		fmt.Sprintf("API Version: %s", version.APIVersion),
		fmt.Sprintf("Operating System: %s", info.OperatingSystem),
		fmt.Sprintf("Containers: %d", len(containers)),
		fmt.Sprintf("Images: %d", len(images)),
		fmt.Sprintf("Server ID: %s", info.ID),
	}
	m.display.PrintInfo(infoItems)
	fmt.Println()
}