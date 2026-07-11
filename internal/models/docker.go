package models

type DockerInfoResponse struct {
	Connected bool `json:"connected"`

	Name              string `json:"name"`
	Platform          string `json:"platform"`
	ServerVersion     string `json:"server_version"`
	APIVersion        string `json:"api_version"`
	MinimumAPIVersion string `json:"minimum_api_version"`

	OperatingSystem string `json:"operating_system"`
	OSType          string `json:"os_type"`
	Architecture    string `json:"architecture"`
	KernelVersion   string `json:"kernel_version"`

	CPUs        int    `json:"cpus"`
	MemoryBytes int64  `json:"memory_bytes"`
	MemoryHuman string `json:"memory_human"`

	Containers        int `json:"containers"`
	ContainersRunning int `json:"containers_running"`
	ContainersPaused  int `json:"containers_paused"`
	ContainersStopped int `json:"containers_stopped"`
	Images            int `json:"images"`

	StorageDriver  string `json:"storage_driver"`
	LoggingDriver  string `json:"logging_driver"`
	DockerRootDir  string `json:"docker_root_dir"`
	DefaultRuntime string `json:"default_runtime"`
}
