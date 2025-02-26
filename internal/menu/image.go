package menu

import (
	"context"
	"fmt"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/internal/domain/image"
	"github.com/Gostatsog/dockerNav/internal/ui"
)

// ImageMenu represents the image menu
type ImageMenu struct {
	dockerClient *client.DockerClient
	ctx          context.Context
	service      *image.Service
}

// NewImageMenu creates a new image menu
func NewImageMenu(ctx context.Context, dockerClient *client.DockerClient) *ImageMenu {
	return &ImageMenu{
		dockerClient: dockerClient,
		ctx:          ctx,
		service:      image.NewService(dockerClient),
	}
}

// Display displays the image menu
func (m *ImageMenu) Display() {
	for {
		options := ui.SelectOptions{
			Label: "Image Management",
			Items: []string{
				"List Images",
				"Pull Image",
				"Remove Image",
				"Build Image",
				"Back",
			},
		}

		_, result, err := ui.Select(options)
		if err != nil {
			fmt.Printf("Prompt failed: %v\n", err)
			return
		}

		switch result {
		case "List Images":
			m.service.ListImages()
		case "Pull Image":
			m.service.PullImage()
		case "Remove Image":
			m.service.RemoveImage()
		case "Build Image":
			m.service.BuildImage()
		case "Back":
			return
		}
	}
}