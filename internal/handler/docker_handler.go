package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"docker-manager-backend/internal/dockerclient"
	"docker-manager-backend/internal/models"
	"docker-manager-backend/internal/response"

	containertype "github.com/moby/moby/api/types/container"
)

type DockerHandler struct {
	dockerClient *dockerclient.Client
}

func NewDockerHandler(
	dockerClient *dockerclient.Client,
) *DockerHandler {
	return &DockerHandler{
		dockerClient: dockerClient,
	}
}

func (h *DockerHandler) Info(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		r.Context(),
		10*time.Second,
	)
	defer cancel()

	if _, err := h.dockerClient.Ping(ctx); err != nil {
		response.Error(
			w,
			http.StatusServiceUnavailable,
			"Docker Engine is unavailable",
		)
		return
	}

	infoResult, err := h.dockerClient.Info(ctx)
	if err != nil {
		response.Error(
			w,
			http.StatusInternalServerError,
			"Cannot read Docker system information",
		)
		return
	}

	versionResult, err := h.dockerClient.ServerVersion(ctx)
	if err != nil {
		response.Error(
			w,
			http.StatusInternalServerError,
			"Cannot read Docker version information",
		)
		return
	}

	info := infoResult.Info

	payload := models.DockerInfoResponse{
		Connected: true,

		Name:              info.Name,
		Platform:          versionResult.Platform.Name,
		ServerVersion:     versionResult.Version,
		APIVersion:        versionResult.APIVersion,
		MinimumAPIVersion: versionResult.MinAPIVersion,

		OperatingSystem: info.OperatingSystem,
		OSType:          info.OSType,
		Architecture:    info.Architecture,
		KernelVersion:   info.KernelVersion,

		CPUs:        info.NCPU,
		MemoryBytes: info.MemTotal,
		MemoryHuman: formatBytes(info.MemTotal),

		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
		Images:            info.Images,

		StorageDriver:  info.Driver,
		LoggingDriver:  info.LoggingDriver,
		DockerRootDir:  info.DockerRootDir,
		DefaultRuntime: info.DefaultRuntime,
	}

	response.JSON(
		w,
		http.StatusOK,
		payload,
	)
}

func (h *DockerHandler) ListContainers(
	w http.ResponseWriter,
	r *http.Request,
) {
	showAll, err := parseAllQuery(r)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"query parameter 'all' must be true or false",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		10*time.Second,
	)
	defer cancel()

	result, err := h.dockerClient.ContainerList(
		ctx,
		showAll,
	)
	if err != nil {
		log.Printf("list Docker containers error: %v", err)

		response.Error(
			w,
			http.StatusServiceUnavailable,
			"Cannot read Docker containers",
		)
		return
	}

	items := make(
		[]models.ContainerResponse,
		0,
		len(result.Items),
	)

	for _, container := range result.Items {
		items = append(
			items,
			mapContainerResponse(container),
		)
	}

	response.JSON(
		w,
		http.StatusOK,
		models.ContainerListResponse{
			Total: len(items),
			Items: items,
		},
	)
}

func parseAllQuery(
	r *http.Request,
) (bool, error) {
	value := r.URL.Query().Get("all")

	if value == "" {
		return true, nil
	}

	return strconv.ParseBool(value)
}

func mapContainerResponse(
	container containertype.Summary,
) models.ContainerResponse {
	ports := make(
		[]models.ContainerPortResponse,
		0,
		len(container.Ports),
	)

	for _, port := range container.Ports {
		var hostPort *uint16

		if port.PublicPort > 0 {
			publicPort := port.PublicPort
			hostPort = &publicPort
		}

		hostIP := ""

		if port.IP.IsValid() {
			hostIP = port.IP.String()
		}

		ports = append(
			ports,
			models.ContainerPortResponse{
				HostIP:        hostIP,
				ContainerPort: port.PrivatePort,
				HostPort:      hostPort,
				Protocol:      port.Type,
			},
		)
	}

	networks := make([]string, 0)

	if container.NetworkSettings != nil {
		networks = make(
			[]string,
			0,
			len(container.NetworkSettings.Networks),
		)

		for networkName := range container.NetworkSettings.Networks {
			networks = append(networks, networkName)
		}

		sort.Strings(networks)
	}

	health := ""

	if container.Health != nil {
		health = string(container.Health.Status)
	}

	return models.ContainerResponse{
		ID:      container.ID,
		ShortID: shortContainerID(container.ID),
		Name:    containerName(container.Names),

		Image:   container.Image,
		ImageID: container.ImageID,
		Command: container.Command,

		State:  string(container.State),
		Status: container.Status,
		Health: health,

		CreatedAt: time.Unix(
			container.Created,
			0,
		).UTC(),

		Ports:       ports,
		Networks:    networks,
		NetworkMode: container.HostConfig.NetworkMode,

		ComposeProject: container.Labels[
			"com.docker.compose.project",
		],
		ComposeService: container.Labels[
			"com.docker.compose.service",
		],

		MountCount: len(container.Mounts),
	}
}

func shortContainerID(id string) string {
	const shortIDLength = 12

	if len(id) <= shortIDLength {
		return id
	}

	return id[:shortIDLength]
}

func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}

	return strings.TrimPrefix(
		names[0],
		"/",
	)
}

func formatBytes(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	divisor := int64(unit)
	exponent := 0

	for value := size / unit; value >= unit; value /= unit {
		divisor *= unit
		exponent++
	}

	units := "KMGTPE"

	return fmt.Sprintf(
		"%.1f %cB",
		float64(size)/float64(divisor),
		units[exponent],
	)
}
