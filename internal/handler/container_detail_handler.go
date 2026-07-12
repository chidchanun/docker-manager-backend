package handler

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"docker-manager-backend/internal/models"
	"docker-manager-backend/internal/response"

	cerrdefs "github.com/containerd/errdefs"
	containertype "github.com/moby/moby/api/types/container"
	mobyclient "github.com/moby/moby/client"
)

const (
	defaultContainerLogTail = 200
	maxContainerLogTail     = 5000
	maxContainerLogBytes    = 2 * 1024 * 1024
)

func (h *DockerHandler) ContainerDetail(
	w http.ResponseWriter,
	r *http.Request,
) {
	containerIdentifier, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		10*time.Second,
	)
	defer cancel()

	result, err := h.dockerClient.ContainerInspect(
		ctx,
		containerIdentifier,
	)
	if err != nil {
		writeDockerReadError(
			w,
			"inspect container",
			containerIdentifier,
			err,
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		mapContainerDetailResponse(result),
	)
}

func (h *DockerHandler) ContainerLogs(
	w http.ResponseWriter,
	r *http.Request,
) {
	containerIdentifier, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}

	tail, err := parseLogTail(r)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"query parameter 'tail' must be between 1 and 5000",
		)
		return
	}

	timestamps, err := parseBooleanQuery(
		r,
		"timestamps",
		false,
	)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"query parameter 'timestamps' must be true or false",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		15*time.Second,
	)
	defer cancel()

	inspectResult, err := h.dockerClient.ContainerInspect(
		ctx,
		containerIdentifier,
	)
	if err != nil {
		writeDockerReadError(
			w,
			"inspect container before reading logs",
			containerIdentifier,
			err,
		)
		return
	}

	container := inspectResult.Container

	logStream, err := h.dockerClient.ContainerLogs(
		ctx,
		containerIdentifier,
		mobyclient.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: timestamps,
			Follow:     false,
			Tail:       strconv.Itoa(tail),
		},
	)
	if err != nil {
		writeDockerReadError(
			w,
			"read container logs",
			containerIdentifier,
			err,
		)
		return
	}
	defer logStream.Close()

	tty := false

	if container.Config != nil {
		tty = container.Config.Tty
	}

	logText, truncated, err := readDockerLogStream(
		logStream,
		tty,
		maxContainerLogBytes,
	)
	if err != nil {
		log.Printf(
			"decode Docker logs error, container=%q: %v",
			containerIdentifier,
			err,
		)

		response.Error(
			w,
			http.StatusInternalServerError,
			"Cannot decode Docker container logs",
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		models.ContainerLogsResponse{
			ContainerID: container.ID,
			Name: strings.TrimPrefix(
				container.Name,
				"/",
			),
			Tail:       tail,
			Timestamps: timestamps,
			Truncated:  truncated,
			Logs:       logText,
		},
	)
}

func parseLogTail(
	r *http.Request,
) (int, error) {
	value := strings.TrimSpace(
		r.URL.Query().Get("tail"),
	)

	if value == "" {
		return defaultContainerLogTail, nil
	}

	tail, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	if tail < 1 || tail > maxContainerLogTail {
		return 0, errors.New(
			"tail value is outside the allowed range",
		)
	}

	return tail, nil
}

func parseBooleanQuery(
	r *http.Request,
	name string,
	defaultValue bool,
) (bool, error) {
	value := strings.TrimSpace(
		r.URL.Query().Get(name),
	)

	if value == "" {
		return defaultValue, nil
	}

	return strconv.ParseBool(value)
}

func readDockerLogStream(
	reader io.Reader,
	tty bool,
	maxBytes int64,
) (string, bool, error) {
	if tty {
		return readRawDockerLogStream(
			reader,
			maxBytes,
		)
	}

	return readMultiplexedDockerLogStream(
		reader,
		maxBytes,
	)
}

func readRawDockerLogStream(
	reader io.Reader,
	maxBytes int64,
) (string, bool, error) {
	limitedReader := io.LimitReader(
		reader,
		maxBytes+1,
	)

	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", false, err
	}

	truncated := int64(len(data)) > maxBytes

	if truncated {
		data = data[:maxBytes]
	}

	return string(data), truncated, nil
}

func readMultiplexedDockerLogStream(
	reader io.Reader,
	maxBytes int64,
) (string, bool, error) {
	var output bytes.Buffer

	header := make([]byte, 8)

	for {
		_, err := io.ReadFull(reader, header)
		if errors.Is(err, io.EOF) {
			break
		}

		if errors.Is(err, io.ErrUnexpectedEOF) {
			return "", false, errors.New(
				"unexpected end of Docker log stream header",
			)
		}

		if err != nil {
			return "", false, err
		}

		frameSize := int64(
			binary.BigEndian.Uint32(header[4:8]),
		)

		if frameSize == 0 {
			continue
		}

		remainingBytes := maxBytes - int64(output.Len())

		if remainingBytes <= 0 {
			return output.String(), true, nil
		}

		bytesToCopy := frameSize

		if bytesToCopy > remainingBytes {
			bytesToCopy = remainingBytes
		}

		_, err = io.CopyN(
			&output,
			reader,
			bytesToCopy,
		)
		if err != nil {
			return "", false, err
		}

		if frameSize > bytesToCopy {
			return output.String(), true, nil
		}
	}

	return output.String(), false, nil
}

func mapContainerDetailResponse(
	result mobyclient.ContainerInspectResult,
) models.ContainerDetailResponse {
	container := result.Container

	payload := models.ContainerDetailResponse{
		ID:           container.ID,
		ShortID:      shortContainerID(container.ID),
		Name:         strings.TrimPrefix(container.Name, "/"),
		CreatedAt:    container.Created,
		ImageID:      container.Image,
		Path:         container.Path,
		Arguments:    safeStringSlice(container.Args),
		Platform:     container.Platform,
		Driver:       container.Driver,
		RestartCount: container.RestartCount,

		Ports:    make([]models.ContainerPortBindingDetailResponse, 0),
		Networks: make([]models.ContainerNetworkDetailResponse, 0),
		Mounts:   make([]models.ContainerMountDetailResponse, 0),
	}

	if container.State != nil {
		payload.State = models.ContainerStateDetailResponse{
			Status:     string(container.State.Status),
			Running:    container.State.Running,
			Paused:     container.State.Paused,
			Restarting: container.State.Restarting,
			OOMKilled:  container.State.OOMKilled,
			Dead:       container.State.Dead,
			PID:        container.State.Pid,
			ExitCode:   container.State.ExitCode,
			Error:      container.State.Error,
			StartedAt:  container.State.StartedAt,
			FinishedAt: container.State.FinishedAt,
		}

		if container.State.Health != nil {
			payload.State.Health = string(
				container.State.Health.Status,
			)

			payload.State.HealthFailingStreak =
				container.State.Health.FailingStreak
		}
	}

	if container.Config != nil {
		payload.Config = models.ContainerConfigDetailResponse{
			Image:      container.Config.Image,
			Hostname:   container.Config.Hostname,
			User:       container.Config.User,
			WorkingDir: container.Config.WorkingDir,
			Entrypoint: redactCommandSecrets(safeStringSlice(
				container.Config.Entrypoint,
			)),
			Command: redactCommandSecrets(safeStringSlice(
				container.Config.Cmd,
			)),
			TTY:        container.Config.Tty,
			StopSignal: container.Config.StopSignal,
		}

		payload.Compose = models.ContainerComposeDetailResponse{
			Project:    container.Config.Labels["com.docker.compose.project"],
			Service:    container.Config.Labels["com.docker.compose.service"],
			ConfigFile: container.Config.Labels["com.docker.compose.project.config_files"],
			WorkingDir: container.Config.Labels["com.docker.compose.project.working_dir"],
		}
	}

	if container.HostConfig != nil {
		payload.HostConfig = models.ContainerHostDetailResponse{
			NetworkMode: string(
				container.HostConfig.NetworkMode,
			),
			LogDriver: container.HostConfig.LogConfig.Type,

			RestartPolicy: string(
				container.HostConfig.RestartPolicy.Name,
			),
			MaximumRestartRetries: container.HostConfig.RestartPolicy.MaximumRetryCount,

			AutoRemove:     container.HostConfig.AutoRemove,
			Privileged:     container.HostConfig.Privileged,
			ReadonlyRootFS: container.HostConfig.ReadonlyRootfs,
			SharedMemory:   container.HostConfig.ShmSize,
			MemoryBytes:    container.HostConfig.Memory,
			NanoCPUs:       container.HostConfig.NanoCPUs,
			PidsLimit:      container.HostConfig.PidsLimit,
		}
	}

	payload.Ports = mapContainerPortBindings(
		container.NetworkSettings,
	)

	payload.Networks = mapContainerNetworks(
		container.NetworkSettings,
	)

	for _, mountPoint := range container.Mounts {
		payload.Mounts = append(
			payload.Mounts,
			models.ContainerMountDetailResponse{
				Type:        string(mountPoint.Type),
				Name:        mountPoint.Name,
				Source:      mountPoint.Source,
				Destination: mountPoint.Destination,
				Driver:      mountPoint.Driver,
				Mode:        mountPoint.Mode,
				ReadWrite:   mountPoint.RW,
				Propagation: string(
					mountPoint.Propagation,
				),
			},
		)
	}

	return payload
}

func mapContainerPortBindings(
	networkSettings *containertype.NetworkSettings,
) []models.ContainerPortBindingDetailResponse {
	items := make(
		[]models.ContainerPortBindingDetailResponse,
		0,
	)

	if networkSettings == nil {
		return items
	}

	portNames := make([]string, 0, len(networkSettings.Ports))
	portLookup := make(map[string]struct {
		Port     uint16
		Protocol string
	})

	for port := range networkSettings.Ports {
		portName := port.String()

		portNames = append(portNames, portName)

		portLookup[portName] = struct {
			Port     uint16
			Protocol string
		}{
			Port:     port.Num(),
			Protocol: string(port.Proto()),
		}
	}

	sort.Strings(portNames)

	for _, portName := range portNames {
		portInfo := portLookup[portName]

		var bindingsFound bool

		for port, bindings := range networkSettings.Ports {
			if port.String() != portName {
				continue
			}

			if len(bindings) == 0 {
				items = append(
					items,
					models.ContainerPortBindingDetailResponse{
						ContainerPort: portInfo.Port,
						Protocol:      portInfo.Protocol,
					},
				)
				bindingsFound = true
				break
			}

			for _, binding := range bindings {
				hostIP := ""

				if binding.HostIP.IsValid() {
					hostIP = binding.HostIP.String()
				}

				items = append(
					items,
					models.ContainerPortBindingDetailResponse{
						ContainerPort: portInfo.Port,
						Protocol:      portInfo.Protocol,
						HostIP:        hostIP,
						HostPort:      binding.HostPort,
					},
				)
			}

			bindingsFound = true
			break
		}

		if !bindingsFound {
			items = append(
				items,
				models.ContainerPortBindingDetailResponse{
					ContainerPort: portInfo.Port,
					Protocol:      portInfo.Protocol,
				},
			)
		}
	}

	return items
}

func mapContainerNetworks(
	networkSettings *containertype.NetworkSettings,
) []models.ContainerNetworkDetailResponse {
	items := make(
		[]models.ContainerNetworkDetailResponse,
		0,
	)

	if networkSettings == nil {
		return items
	}

	networkNames := make(
		[]string,
		0,
		len(networkSettings.Networks),
	)

	for networkName := range networkSettings.Networks {
		networkNames = append(
			networkNames,
			networkName,
		)
	}

	sort.Strings(networkNames)

	for _, networkName := range networkNames {
		endpoint := networkSettings.Networks[networkName]
		if endpoint == nil {
			continue
		}

		ipAddress := ""
		gateway := ""
		macAddress := ""

		if endpoint.IPAddress.IsValid() {
			ipAddress = endpoint.IPAddress.String()
		}

		if endpoint.Gateway.IsValid() {
			gateway = endpoint.Gateway.String()
		}

		if len(endpoint.MacAddress) > 0 {
			macAddress = endpoint.MacAddress.String()
		}

		items = append(
			items,
			models.ContainerNetworkDetailResponse{
				Name:       networkName,
				NetworkID:  endpoint.NetworkID,
				EndpointID: endpoint.EndpointID,
				IPAddress:  ipAddress,
				PrefixSize: endpoint.IPPrefixLen,
				Gateway:    gateway,
				MACAddress: macAddress,
				Aliases:    safeStringSlice(endpoint.Aliases),
				DNSNames:   safeStringSlice(endpoint.DNSNames),
			},
		)
	}

	return items
}

func safeStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}

func writeDockerReadError(
	w http.ResponseWriter,
	operation string,
	containerIdentifier string,
	err error,
) {
	log.Printf(
		"Docker %s error, container=%q: %v",
		operation,
		containerIdentifier,
		err,
	)

	switch {
	case errors.Is(err, context.Canceled),
		cerrdefs.IsCanceled(err):
		response.Error(
			w,
			http.StatusRequestTimeout,
			"Request was cancelled",
		)

	case errors.Is(err, context.DeadlineExceeded),
		cerrdefs.IsDeadlineExceeded(err):
		response.Error(
			w,
			http.StatusGatewayTimeout,
			"Docker request timed out",
		)

	case cerrdefs.IsNotFound(err):
		response.Error(
			w,
			http.StatusNotFound,
			"Container not found",
		)

	case cerrdefs.IsInvalidArgument(err):
		response.Error(
			w,
			http.StatusBadRequest,
			"Invalid Docker container parameter",
		)

	case cerrdefs.IsPermissionDenied(err):
		response.Error(
			w,
			http.StatusForbidden,
			"Docker access is forbidden",
		)

	case cerrdefs.IsUnavailable(err):
		response.Error(
			w,
			http.StatusServiceUnavailable,
			"Docker Engine is unavailable",
		)

	default:
		response.Error(
			w,
			http.StatusInternalServerError,
			"Cannot read Docker container information",
		)
	}
}

func redactCommandSecrets(values []string) []string {
	redacted := append([]string(nil), values...)
	secretFlags := map[string]bool{"--token": true, "--password": true, "--secret": true, "--api-key": true, "--apikey": true}
	for index, value := range redacted {
		lower := strings.ToLower(value)
		if index > 0 && secretFlags[strings.ToLower(redacted[index-1])] {
			redacted[index] = "[REDACTED]"
			continue
		}
		for flag := range secretFlags {
			if strings.HasPrefix(lower, flag+"=") {
				redacted[index] = value[:len(flag)+1] + "[REDACTED]"
				break
			}
		}
	}
	return redacted
}
