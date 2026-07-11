package models

import "time"

type ContainerPortResponse struct {
	HostIP        string  `json:"host_ip,omitempty"`
	ContainerPort uint16  `json:"container_port"`
	HostPort      *uint16 `json:"host_port,omitempty"`
	Protocol      string  `json:"protocol"`
}

type ContainerResponse struct {
	ID      string `json:"id"`
	ShortID string `json:"short_id"`
	Name    string `json:"name"`

	Image   string `json:"image"`
	ImageID string `json:"image_id"`
	Command string `json:"command"`

	State  string `json:"state"`
	Status string `json:"status"`
	Health string `json:"health,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	Ports       []ContainerPortResponse `json:"ports"`
	Networks    []string                `json:"networks"`
	NetworkMode string                  `json:"network_mode"`

	ComposeProject string `json:"compose_project,omitempty"`
	ComposeService string `json:"compose_service,omitempty"`

	MountCount int `json:"mount_count"`
}

type ContainerListResponse struct {
	Total int                 `json:"total"`
	Items []ContainerResponse `json:"items"`
}