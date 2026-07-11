package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"docker-manager-backend/internal/models"
	"docker-manager-backend/internal/response"

	cerrdefs "github.com/containerd/errdefs"
	mobyclient "github.com/moby/moby/client"
)

const (
	defaultContainerActionTimeoutSeconds = 10
	maxContainerActionTimeoutSeconds     = 60
)

func (h *DockerHandler) StartContainer(
	w http.ResponseWriter,
	r *http.Request,
) {
	containerIdentifier, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		20*time.Second,
	)
	defer cancel()

	inspectResult, err := h.dockerClient.ContainerInspect(
		ctx,
		containerIdentifier,
	)
	if err != nil {
		writeDockerActionError(
			w,
			"start",
			containerIdentifier,
			err,
		)
		return
	}

	containerID, containerName, currentState, running :=
		containerActionData(inspectResult)

	if running {
		writeContainerActionResponse(
			w,
			"start",
			containerID,
			containerName,
			currentState,
			false,
			nil,
			"Container is already running",
		)
		return
	}

	err = h.dockerClient.ContainerStart(
		ctx,
		containerIdentifier,
	)
	if err != nil {
		if cerrdefs.IsNotModified(err) {
			writeContainerActionResponse(
				w,
				"start",
				containerID,
				containerName,
				currentState,
				false,
				nil,
				"Container is already running",
			)
			return
		}

		writeDockerActionError(
			w,
			"start",
			containerIdentifier,
			err,
		)
		return
	}

	newState := h.refreshContainerState(
		ctx,
		containerIdentifier,
		"running",
	)

	writeContainerActionResponse(
		w,
		"start",
		containerID,
		containerName,
		newState,
		true,
		nil,
		"Container started successfully",
	)
}

func (h *DockerHandler) StopContainer(
	w http.ResponseWriter,
	r *http.Request,
) {
	containerIdentifier, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}

	timeoutSeconds, err := parseContainerActionTimeout(r)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"query parameter 'timeout' must be between 0 and 60 seconds",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		time.Duration(timeoutSeconds+10)*time.Second,
	)
	defer cancel()

	inspectResult, err := h.dockerClient.ContainerInspect(
		ctx,
		containerIdentifier,
	)
	if err != nil {
		writeDockerActionError(
			w,
			"stop",
			containerIdentifier,
			err,
		)
		return
	}

	containerID, containerName, currentState, running :=
		containerActionData(inspectResult)

	if !running {
		writeContainerActionResponse(
			w,
			"stop",
			containerID,
			containerName,
			currentState,
			false,
			&timeoutSeconds,
			"Container is already stopped",
		)
		return
	}

	err = h.dockerClient.ContainerStop(
		ctx,
		containerIdentifier,
		timeoutSeconds,
	)
	if err != nil {
		if  cerrdefs.IsNotModified(err) {
			writeContainerActionResponse(
				w,
				"stop",
				containerID,
				containerName,
				currentState,
				false,
				&timeoutSeconds,
				"Container is already stopped",
			)
			return
		}

		writeDockerActionError(
			w,
			"stop",
			containerIdentifier,
			err,
		)
		return
	}

	newState := h.refreshContainerState(
		ctx,
		containerIdentifier,
		"exited",
	)

	writeContainerActionResponse(
		w,
		"stop",
		containerID,
		containerName,
		newState,
		true,
		&timeoutSeconds,
		"Container stopped successfully",
	)
}

func (h *DockerHandler) RestartContainer(
	w http.ResponseWriter,
	r *http.Request,
) {
	containerIdentifier, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}

	timeoutSeconds, err := parseContainerActionTimeout(r)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"query parameter 'timeout' must be between 0 and 60 seconds",
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		time.Duration(timeoutSeconds+10)*time.Second,
	)
	defer cancel()

	inspectResult, err := h.dockerClient.ContainerInspect(
		ctx,
		containerIdentifier,
	)
	if err != nil {
		writeDockerActionError(
			w,
			"restart",
			containerIdentifier,
			err,
		)
		return
	}

	containerID, containerName, _, _ :=
		containerActionData(inspectResult)

	err = h.dockerClient.ContainerRestart(
		ctx,
		containerIdentifier,
		timeoutSeconds,
	)
	if err != nil {
		writeDockerActionError(
			w,
			"restart",
			containerIdentifier,
			err,
		)
		return
	}

	newState := h.refreshContainerState(
		ctx,
		containerIdentifier,
		"running",
	)

	writeContainerActionResponse(
		w,
		"restart",
		containerID,
		containerName,
		newState,
		true,
		&timeoutSeconds,
		"Container restarted successfully",
	)
}

func containerIdentifierFromRequest(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	containerIdentifier := strings.TrimSpace(
		r.PathValue("id"),
	)

	if containerIdentifier == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"Container ID or name is required",
		)
		return "", false
	}

	return containerIdentifier, true
}

func parseContainerActionTimeout(
	r *http.Request,
) (int, error) {
	value := strings.TrimSpace(
		r.URL.Query().Get("timeout"),
	)

	if value == "" {
		return defaultContainerActionTimeoutSeconds, nil
	}

	timeoutSeconds, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	if timeoutSeconds < 0 ||
		timeoutSeconds > maxContainerActionTimeoutSeconds {
		return 0, errors.New(
			"timeout is outside the allowed range",
		)
	}

	return timeoutSeconds, nil
}

func containerActionData(
	result mobyclient.ContainerInspectResult,
) (
	containerID string,
	containerName string,
	state string,
	running bool,
) {
	containerID = result.Container.ID
	containerName = strings.TrimPrefix(
		result.Container.Name,
		"/",
	)
	state = "unknown"

	if result.Container.State != nil {
		state = string(result.Container.State.Status)
		running = result.Container.State.Running
	}

	return
}

func (h *DockerHandler) refreshContainerState(
	ctx context.Context,
	containerIdentifier string,
	fallbackState string,
) string {
	inspectResult, err := h.dockerClient.ContainerInspect(
		ctx,
		containerIdentifier,
	)
	if err != nil {
		log.Printf(
			"refresh state for container %q error: %v",
			containerIdentifier,
			err,
		)

		return fallbackState
	}

	if inspectResult.Container.State == nil {
		return fallbackState
	}

	return string(
		inspectResult.Container.State.Status,
	)
}

func writeContainerActionResponse(
	w http.ResponseWriter,
	action string,
	containerID string,
	containerName string,
	state string,
	changed bool,
	timeoutSeconds *int,
	message string,
) {
	response.JSON(
		w,
		http.StatusOK,
		models.ContainerActionResponse{
			Success: true,
			Action:  action,

			ContainerID: containerID,
			Name:        containerName,
			State:       state,

			Changed:        changed,
			TimeoutSeconds: timeoutSeconds,

			Message:     message,
			PerformedAt: time.Now().UTC(),
		},
	)
}

func writeDockerActionError(
	w http.ResponseWriter,
	action string,
	containerIdentifier string,
	err error,
) {
	log.Printf(
		"Docker container %s error, container=%q: %v",
		action,
		containerIdentifier,
		err,
	)

	switch {
	case errors.Is(err, context.Canceled):
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
			"Docker action timed out",
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
			"Docker action is forbidden",
		)

	case cerrdefs.IsConflict(err):
		response.Error(
			w,
			http.StatusConflict,
			"Container state conflicts with the requested action",
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
			"Cannot perform Docker container action",
		)
	}
}