package client

import (
	"context"

	"github.com/docker/docker/client"
)

// DockerClient wraps the Docker client SDK
type DockerClient struct {
	Client *client.Client
	Ctx    context.Context
}

// NewDockerClient creates a new Docker client
func NewDockerClient(ctx context.Context) (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerClient{
		Client: cli,
		Ctx:    ctx,
	}, nil
}

// Close closes the Docker client
func (d *DockerClient) Close() error {
	if d.Client != nil {
		return d.Client.Close()
	}
	return nil
}