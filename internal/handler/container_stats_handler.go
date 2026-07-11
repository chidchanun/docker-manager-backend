package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	appmetrics "docker-manager-backend/internal/metrics"
	"docker-manager-backend/internal/models"
	"docker-manager-backend/internal/response"

	containertype "github.com/moby/moby/api/types/container"
)

func (h *DockerHandler) ContainerStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	listed, err := h.dockerClient.ContainerList(ctx, false)
	if err != nil {
		response.Error(w, http.StatusServiceUnavailable, "Cannot list running containers")
		return
	}

	items := make([]models.ContainerStatsResponse, len(listed.Items))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for index, summary := range listed.Items {
		wg.Add(1)
		go func(i int, id string, names []string) {
			defer wg.Done()
			result, statsErr := h.dockerClient.ContainerStats(ctx, id)
			if statsErr != nil {
				log.Printf("container stats %s: %v", id, statsErr)
				return
			}
			defer result.Body.Close()
			var stats containertype.StatsResponse
			if statsErr = json.NewDecoder(result.Body).Decode(&stats); statsErr != nil {
				log.Printf("decode stats %s: %v", id, statsErr)
				return
			}
			item := mapContainerStats(stats)
			item.ID = id
			if len(names) > 0 {
				item.Name = strings.TrimPrefix(names[0], "/")
			}
			mu.Lock()
			items[i] = item
			mu.Unlock()
		}(index, summary.ID, summary.Names)
	}
	wg.Wait()

	valid := items[:0]
	total := models.ContainerStatsResponse{Name: "All containers"}
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		valid = append(valid, item)
		appmetrics.CPU.WithLabelValues(item.ID, item.Name).Set(item.CPUPercent)
		appmetrics.Memory.WithLabelValues(item.ID, item.Name).Set(float64(item.MemoryUsage))
		appmetrics.NetworkRX.WithLabelValues(item.ID, item.Name).Set(float64(item.NetworkRX))
		appmetrics.NetworkTX.WithLabelValues(item.ID, item.Name).Set(float64(item.NetworkTX))
		total.CPUPercent += item.CPUPercent
		total.MemoryUsage += item.MemoryUsage
		// Containers without an explicit memory limit report the shared
		// Docker host/VM limit. Do not add that same pool once per container.
		if item.MemoryLimit > total.MemoryLimit {
			total.MemoryLimit = item.MemoryLimit
		}
		total.DiskRead += item.DiskRead
		total.DiskWrite += item.DiskWrite
		total.NetworkRX += item.NetworkRX
		total.NetworkTX += item.NetworkTX
	}
	if total.MemoryLimit > 0 {
		total.MemoryPercent = float64(total.MemoryUsage) / float64(total.MemoryLimit) * 100
	}
	response.JSON(w, http.StatusOK, models.ContainerStatsListResponse{CollectedAt: time.Now().UTC(), Total: total, Items: valid})
}

func mapContainerStats(stats containertype.StatsResponse) models.ContainerStatsResponse {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cores := stats.CPUStats.OnlineCPUs
	if cores == 0 {
		cores = uint32(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta >= 0 {
		cpuPercent = cpuDelta / systemDelta * float64(cores) * 100
	} else if !stats.PreRead.IsZero() {
		elapsed := float64(stats.Read.Sub(stats.PreRead).Nanoseconds())
		if elapsed > 0 {
			cpuPercent = cpuDelta * 10000 / elapsed
		}
	}
	memory := stats.MemoryStats.Usage
	if stats.MemoryStats.PrivateWorkingSet > 0 {
		memory = stats.MemoryStats.PrivateWorkingSet
	}
	result := models.ContainerStatsResponse{CPUPercent: cpuPercent, MemoryUsage: memory, MemoryLimit: stats.MemoryStats.Limit}
	if result.MemoryLimit > 0 {
		result.MemoryPercent = float64(memory) / float64(result.MemoryLimit) * 100
	}
	for _, entry := range stats.BlkioStats.IoServiceBytesRecursive {
		if strings.EqualFold(entry.Op, "read") {
			result.DiskRead += entry.Value
		}
		if strings.EqualFold(entry.Op, "write") {
			result.DiskWrite += entry.Value
		}
	}
	result.DiskRead += stats.StorageStats.ReadSizeBytes
	result.DiskWrite += stats.StorageStats.WriteSizeBytes
	for _, network := range stats.Networks {
		result.NetworkRX += network.RxBytes
		result.NetworkTX += network.TxBytes
	}
	return result
}
