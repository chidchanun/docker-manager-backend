package dockerclient

import (
	"context"
	"fmt"
	"time"

	mobyclient "github.com/moby/moby/client"
)

type Client struct {
	api *mobyclient.Client
}

func New() (*Client, error) {
	apiClient, err := mobyclient.New(
		mobyclient.FromEnv,
		mobyclient.WithUserAgent("docker-manager-backend/0.1.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("create Docker client: %w", err)
	}

	dockerClient := &Client{
		api: apiClient,
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if _, err := dockerClient.Ping(ctx); err != nil {
		dockerClient.Close()

		return nil, fmt.Errorf(
			"cannot connect to Docker Engine: %w",
			err,
		)
	}

	return dockerClient, nil
}

func (c *Client) Ping(
	ctx context.Context,
) (mobyclient.PingResult, error) {
	return c.api.Ping(
		ctx,
		mobyclient.PingOptions{},
	)
}

func (c *Client) Info(
	ctx context.Context,
) (mobyclient.SystemInfoResult, error) {
	return c.api.Info(
		ctx,
		mobyclient.InfoOptions{},
	)
}

func (c *Client) ServerVersion(
	ctx context.Context,
) (mobyclient.ServerVersionResult, error) {
	return c.api.ServerVersion(
		ctx,
		mobyclient.ServerVersionOptions{},
	)
}

func (c *Client) ContainerList(
	ctx context.Context,
	all bool,
) (mobyclient.ContainerListResult, error) {
	return c.api.ContainerList(
		ctx,
		mobyclient.ContainerListOptions{
			All: all,
		},
	)
}

func (c *Client) ContainerStats(
	ctx context.Context,
	containerID string,
) (mobyclient.ContainerStatsResult, error) {
	return c.api.ContainerStats(ctx, containerID, mobyclient.ContainerStatsOptions{
		Stream:                false,
		IncludePreviousSample: true,
	})
}

func (c *Client) ContainerInspect(
	ctx context.Context,
	containerID string,
) (mobyclient.ContainerInspectResult, error) {
	return c.api.ContainerInspect(
		ctx,
		containerID,
		mobyclient.ContainerInspectOptions{
			Size: false,
		},
	)
}

func (c *Client) ContainerStart(
	ctx context.Context,
	containerID string,
) error {
	_, err := c.api.ContainerStart(
		ctx,
		containerID,
		mobyclient.ContainerStartOptions{},
	)

	return err
}

func (c *Client) ContainerStop(
	ctx context.Context,
	containerID string,
	timeoutSeconds int,
) error {
	_, err := c.api.ContainerStop(
		ctx,
		containerID,
		mobyclient.ContainerStopOptions{
			Timeout: &timeoutSeconds,
		},
	)

	return err
}

func (c *Client) ContainerRestart(
	ctx context.Context,
	containerID string,
	timeoutSeconds int,
) error {
	_, err := c.api.ContainerRestart(
		ctx,
		containerID,
		mobyclient.ContainerRestartOptions{
			Timeout: &timeoutSeconds,
		},
	)

	return err
}

func (c *Client) ContainerPause(ctx context.Context, id string) error {
	_, err := c.api.ContainerPause(ctx, id, mobyclient.ContainerPauseOptions{})
	return err
}
func (c *Client) ContainerUnpause(ctx context.Context, id string) error {
	_, err := c.api.ContainerUnpause(ctx, id, mobyclient.ContainerUnpauseOptions{})
	return err
}
func (c *Client) ContainerKill(ctx context.Context, id string) error {
	_, err := c.api.ContainerKill(ctx, id, mobyclient.ContainerKillOptions{Signal: "SIGKILL"})
	return err
}
func (c *Client) ContainerRemove(ctx context.Context, id string) error {
	_, err := c.api.ContainerRemove(ctx, id, mobyclient.ContainerRemoveOptions{Force: false, RemoveVolumes: false})
	return err
}

func (c *Client) ContainerLogs(
	ctx context.Context,
	containerID string,
	options mobyclient.ContainerLogsOptions,
) (mobyclient.ContainerLogsResult, error) {
	return c.api.ContainerLogs(
		ctx,
		containerID,
		options,
	)
}

func (c *Client) Close() {
	if c == nil || c.api == nil {
		return
	}

	_ = c.api.Close()
}
