package app

import (
	"context"
	"fmt"

	"github.com/Gostasog/dockerNav/internal/client"
	"github.com/Gostasog/dockerNav/internal/menu"
)

// App represents the main application
type App struct {
	dockerClient *client.DockerClient
	ctx          context.Context
}

// NewApp creates a new instance of the application
func NewApp() (*App, error) {
	// Initialize context
	ctx := context.Background()

	// Initialize Docker client
	dockerClient, err := client.NewDockerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing Docker client: %w", err)
	}

	return &App{
		dockerClient: dockerClient,
		ctx:          ctx,
	}, nil
}

// StartMainMenu starts the main menu
func (a *App) StartMainMenu() {
	mainMenu := menu.NewMainMenu(a.ctx, a.dockerClient)
	mainMenu.Display()
}