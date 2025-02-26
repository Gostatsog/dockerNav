package image

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/internal/ui"
	"github.com/Gostatsog/dockerNav/pkg/formatter"
)

// Service provides image operations
type Service struct {
	dockerClient *client.DockerClient
}

// NewService creates a new image service
func NewService(dockerClient *client.DockerClient) *Service {
	return &Service{
		dockerClient: dockerClient,
	}
}

// ListImages displays a list of Docker images
func (s *Service) ListImages() {
	images, err := s.dockerClient.Client.ImageList(s.dockerClient.Ctx, image.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing images: %v\n", err)
		return
	}

	if len(images) == 0 {
		fmt.Println("No images found")
		ui.WaitForEnter()
		return
	}

	// Prepare table data
	rows := make([][]string, 0, len(images))
	for _, img := range images {
		// Handle images with no repository/tag
		repo := "<none>"
		tag := "<none>"
		
		if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
			parts := strings.Split(img.RepoTags[0], ":")
			if len(parts) > 1 {
				repo = parts[0]
				tag = parts[1]
			}
		}
		
		// Format created time
		createdTime := time.Unix(img.Created, 0)
		created := formatter.FormatTime(createdTime)
		
		// Format size
		size := formatter.FormatSize(float64(img.Size))
		
		rows = append(rows, []string{
			repo,
			tag,
			img.ID[7:19], // shortened ID
			created,
			size,
		})
	}

	// Display table
	tableOptions := ui.TableOptions{
		Headers: []string{"REPOSITORY", "TAG", "IMAGE ID", "CREATED", "SIZE"},
		Rows:    rows,
		HeaderColors: ui.DefaultHeaderColors(5),
	}

	ui.RenderTable(tableOptions)
	ui.WaitForEnter()
}

// PullImage pulls a Docker image
func (s *Service) PullImage() {
	promptOptions := ui.PromptOptions{
		Label: "Enter image name (e.g., ubuntu:latest)",
	}

	imgName, err := ui.Prompt(promptOptions)
	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		return
	}

	fmt.Printf("Pulling image %s...\n", imgName)
	reader, err := s.dockerClient.Client.ImagePull(s.dockerClient.Ctx, imgName, image.PullOptions{})
	if err != nil {
		fmt.Printf("Error pulling image: %v\n", err)
		ui.WaitForEnter()
		return
	}
	defer reader.Close()

	// Read and display the output
	buf := make([]byte, 1024)
	for {
		_, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error reading pull output: %v\n", err)
			break
		}
	}

	fmt.Printf("Image %s pulled successfully\n", imgName)
	ui.WaitForEnter()
}

// RemoveImage removes a Docker image
func (s *Service) RemoveImage() {
	images, err := s.dockerClient.Client.ImageList(s.dockerClient.Ctx, image.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing images: %v\n", err)
		return
	}

	if len(images) == 0 {
		fmt.Println("No images found")
		ui.WaitForEnter()
		return
	}

	var imageOptions []string
	for _, img := range images {
		if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
			imageOptions = append(imageOptions, img.RepoTags[0])
		} else {
			imageOptions = append(imageOptions, img.ID[7:19])
		}
	}
	imageOptions = append(imageOptions, "Back")

	selectOptions := ui.SelectOptions{
		Label: "Select Image to Remove",
		Items: imageOptions,
		Size:  10,
	}

	_, result, err := ui.Select(selectOptions)
	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		return
	}

	if result == "Back" {
		return
	}

	// Confirm action
	confirmed, err := ui.Confirm(fmt.Sprintf("Remove image %s?", result))
	if err != nil {
		fmt.Printf("Error confirming action: %v\n", err)
		return
	}

	if !confirmed {
		return
	}

	// Remove image
	_, err = s.dockerClient.Client.ImageRemove(s.dockerClient.Ctx, result, image.RemoveOptions{})
	if err != nil {
		fmt.Printf("Error removing image: %v\n", err)
		ui.WaitForEnter()
		return
	}

	fmt.Printf("Image %s removed successfully\n", result)
	ui.WaitForEnter()
}

// BuildImage builds a Docker image from a Dockerfile
func (s *Service) BuildImage() {
	// Prompt for Dockerfile path
	pathPromptOptions := ui.PromptOptions{
		Label:   "Path to Dockerfile directory",
		Default: ".",
	}

	path, err := ui.Prompt(pathPromptOptions)
	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		return
	}

	// Prompt for tag
	tagPromptOptions := ui.PromptOptions{
		Label: "Tag for the image (e.g., myapp:latest)",
	}

	tag, err := ui.Prompt(tagPromptOptions)
	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		return
	}

	fmt.Printf("Building image from %s with tag %s...\n", path, tag)
	
	// Use docker build command as it handles context and build process better
	cmd := exec.Command("docker", "build", "-t", tag, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	
	if err != nil {
		fmt.Printf("Error building image: %v\n", err)
		ui.WaitForEnter()
		return
	}

	fmt.Printf("Image %s built successfully\n", tag)
	ui.WaitForEnter()
}