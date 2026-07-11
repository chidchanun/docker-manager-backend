package models

import "time"

type ContainerActionResponse struct {
	Success bool   `json:"success"`
	Action  string `json:"action"`

	ContainerID string `json:"container_id"`
	Name        string `json:"name"`
	State       string `json:"state"`

	Changed        bool `json:"changed"`
	TimeoutSeconds *int `json:"timeout_seconds,omitempty"`

	Message     string    `json:"message"`
	PerformedAt time.Time `json:"performed_at"`
}