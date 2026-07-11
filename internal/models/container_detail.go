package models

type ContainerStateDetailResponse struct {
	Status     string `json:"status"`
	Running    bool   `json:"running"`
	Paused     bool   `json:"paused"`
	Restarting bool   `json:"restarting"`
	OOMKilled  bool   `json:"oom_killed"`
	Dead       bool   `json:"dead"`

	PID      int    `json:"pid"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`

	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`

	Health              string `json:"health,omitempty"`
	HealthFailingStreak int    `json:"health_failing_streak,omitempty"`
}

type ContainerConfigDetailResponse struct {
	Image      string   `json:"image"`
	Hostname   string   `json:"hostname"`
	User       string   `json:"user,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
	Entrypoint []string `json:"entrypoint"`
	Command    []string `json:"command"`

	TTY        bool   `json:"tty"`
	StopSignal string `json:"stop_signal,omitempty"`
}

type ContainerHostDetailResponse struct {
	NetworkMode string `json:"network_mode"`
	LogDriver   string `json:"log_driver"`

	RestartPolicy         string `json:"restart_policy"`
	MaximumRestartRetries int    `json:"maximum_restart_retries"`

	AutoRemove     bool  `json:"auto_remove"`
	Privileged     bool  `json:"privileged"`
	ReadonlyRootFS bool  `json:"readonly_root_fs"`
	SharedMemory   int64 `json:"shared_memory_bytes"`
}

type ContainerPortBindingDetailResponse struct {
	ContainerPort uint16 `json:"container_port"`
	Protocol      string `json:"protocol"`

	HostIP   string `json:"host_ip,omitempty"`
	HostPort string `json:"host_port,omitempty"`
}

type ContainerNetworkDetailResponse struct {
	Name       string `json:"name"`
	NetworkID  string `json:"network_id"`
	EndpointID string `json:"endpoint_id"`

	IPAddress  string `json:"ip_address,omitempty"`
	PrefixSize int    `json:"prefix_size,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`

	Aliases  []string `json:"aliases"`
	DNSNames []string `json:"dns_names"`
}

type ContainerMountDetailResponse struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Driver      string `json:"driver,omitempty"`
	Mode        string `json:"mode,omitempty"`

	ReadWrite   bool   `json:"read_write"`
	Propagation string `json:"propagation,omitempty"`
}

type ContainerComposeDetailResponse struct {
	Project    string `json:"project,omitempty"`
	Service    string `json:"service,omitempty"`
	ConfigFile string `json:"config_file,omitempty"`
	WorkingDir string `json:"working_dir,omitempty"`
}

type ContainerDetailResponse struct {
	ID      string `json:"id"`
	ShortID string `json:"short_id"`
	Name    string `json:"name"`

	CreatedAt   string `json:"created_at"`
	ImageID     string `json:"image_id"`
	Path        string `json:"path"`
	Arguments   []string `json:"arguments"`
	Platform    string `json:"platform"`
	Driver      string `json:"driver"`
	RestartCount int   `json:"restart_count"`

	State      ContainerStateDetailResponse  `json:"state"`
	Config     ContainerConfigDetailResponse `json:"config"`
	HostConfig ContainerHostDetailResponse   `json:"host_config"`

	Ports    []ContainerPortBindingDetailResponse `json:"ports"`
	Networks []ContainerNetworkDetailResponse     `json:"networks"`
	Mounts   []ContainerMountDetailResponse       `json:"mounts"`

	Compose ContainerComposeDetailResponse `json:"compose"`
}

type ContainerLogsResponse struct {
	ContainerID string `json:"container_id"`
	Name        string `json:"name"`

	Tail       int  `json:"tail"`
	Timestamps bool `json:"timestamps"`
	Truncated  bool `json:"truncated"`

	Logs string `json:"logs"`
}