package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"docker-manager-backend/internal/auth"
)

func (h *DockerHandler) PauseContainer(w http.ResponseWriter, r *http.Request) {
	h.simpleContainerAction(w, r, "pause", h.dockerClient.ContainerPause, "paused", "Container paused successfully")
}
func (h *DockerHandler) UnpauseContainer(w http.ResponseWriter, r *http.Request) {
	h.simpleContainerAction(w, r, "unpause", h.dockerClient.ContainerUnpause, "running", "Container unpaused successfully")
}
func (h *DockerHandler) KillContainer(w http.ResponseWriter, r *http.Request) {
	h.simpleContainerAction(w, r, "kill", h.dockerClient.ContainerKill, "exited", "Container killed successfully")
}
func (h *DockerHandler) RemoveContainer(w http.ResponseWriter, r *http.Request) {
	h.simpleContainerAction(w, r, "remove", h.dockerClient.ContainerRemove, "removed", "Container removed successfully")
}

func (h *DockerHandler) simpleContainerAction(w http.ResponseWriter, r *http.Request, action string, operation func(context.Context, string) error, fallbackState, message string) {
	id, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	inspected, err := h.dockerClient.ContainerInspect(ctx, id)
	if err != nil {
		writeDockerActionError(w, action, id, err)
		return
	}
	containerID, name, _, _ := containerActionData(inspected)
	if err = operation(ctx, id); err != nil {
		writeDockerActionError(w, action, id, err)
		return
	}
	state := fallbackState
	if action != "remove" {
		state = h.refreshContainerState(ctx, id, fallbackState)
	}
	user, _ := auth.UserFromContext(r.Context())
	log.Printf("audit action=%s container=%q id=%q user=%q remote=%q", action, name, containerID, user.Email, r.RemoteAddr)
	writeContainerActionResponse(w, action, containerID, name, state, true, nil, message)
}
