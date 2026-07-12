package handler

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"docker-manager-backend/internal/response"
	cerrdefs "github.com/containerd/errdefs"
	containertype "github.com/moby/moby/api/types/container"
)

type containerPolicyRequest struct {
	RestartPolicy     string  `json:"restart_policy"`
	MaximumRetryCount int     `json:"maximum_retry_count"`
	CPUs              float64 `json:"cpus"`
	MemoryBytes       int64   `json:"memory_bytes"`
	PidsLimit         int64   `json:"pids_limit"`
}

func (h *DockerHandler) UpdateContainerPolicy(w http.ResponseWriter, r *http.Request) {
	id, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request containerPolicyRequest
	if err := decoder.Decode(&request); err != nil {
		response.Error(w, 400, "Invalid policy request")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.Error(w, 400, "Request must contain one JSON object")
		return
	}
	allowed := map[string]bool{"no": true, "always": true, "unless-stopped": true, "on-failure": true}
	if !allowed[request.RestartPolicy] || request.CPUs < 0 || request.CPUs > 256 || request.MemoryBytes < 0 || request.PidsLimit < -1 {
		response.Error(w, 400, "Invalid container policy values")
		return
	}
	if request.RestartPolicy != "on-failure" {
		request.MaximumRetryCount = 0
	}
	pids := request.PidsLimit
	resources := &containertype.Resources{
		NanoCPUs: int64(request.CPUs * 1e9),
		Memory:   request.MemoryBytes,
		// Equal values disable additional swap and satisfy Docker's
		// requirement to update memory and memory-swap together.
		MemorySwap: request.MemoryBytes,
		PidsLimit:  &pids,
	}
	policy := &containertype.RestartPolicy{Name: containertype.RestartPolicyMode(request.RestartPolicy), MaximumRetryCount: request.MaximumRetryCount}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	info, infoErr := h.dockerClient.Info(ctx)
	if infoErr != nil {
		writeDockerActionError(w, "read host limits", id, infoErr)
		return
	}
	if request.CPUs > float64(info.Info.NCPU) {
		response.Error(w, http.StatusBadRequest, "CPU limit exceeds Docker host capacity")
		return
	}
	if request.MemoryBytes > info.Info.MemTotal {
		response.Error(w, http.StatusBadRequest, "Memory limit exceeds Docker host capacity")
		return
	}
	result, err := h.dockerClient.ContainerUpdate(ctx, id, resources, policy)
	if err != nil {
		log.Printf("update container policy error, container=%q: %v", id, err)
		if cerrdefs.IsConflict(err) {
			response.Error(w, http.StatusConflict, "Docker policy conflict: "+err.Error())
			return
		}
		writeDockerActionError(w, "policy", id, err)
		return
	}
	response.JSON(w, 200, map[string]any{"success": true, "message": "Container policy updated successfully", "warnings": result.Warnings})
}
