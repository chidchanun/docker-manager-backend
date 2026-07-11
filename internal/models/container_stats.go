package models

import "time"

type ContainerStatsResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage"`
	MemoryLimit   uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskRead      uint64  `json:"disk_read"`
	DiskWrite     uint64  `json:"disk_write"`
	NetworkRX     uint64  `json:"network_rx"`
	NetworkTX     uint64  `json:"network_tx"`
}

type ContainerStatsListResponse struct {
	CollectedAt time.Time                `json:"collected_at"`
	Total       ContainerStatsResponse   `json:"total"`
	Items       []ContainerStatsResponse `json:"items"`
}
