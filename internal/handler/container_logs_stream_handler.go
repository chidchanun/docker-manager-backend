package handler

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	mobyclient "github.com/moby/moby/client"
)

func (h *DockerHandler) ContainerLogsStream(w http.ResponseWriter, r *http.Request) {
	id, ok := containerIdentifierFromRequest(w, r)
	if !ok {
		return
	}
	tail, err := parseLogTail(r)
	if err != nil {
		http.Error(w, "invalid tail", 400)
		return
	}
	timestamps, err := parseBooleanQuery(r, "timestamps", false)
	if err != nil {
		http.Error(w, "invalid timestamps", 400)
		return
	}
	inspect, err := h.dockerClient.ContainerInspect(r.Context(), id)
	if err != nil {
		writeDockerReadError(w, "inspect before streaming logs", id, err)
		return
	}
	stream, err := h.dockerClient.ContainerLogs(r.Context(), id, mobyclient.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Timestamps: timestamps, Follow: true, Tail: fmt.Sprint(tail)})
	if err != nil {
		writeDockerReadError(w, "stream logs", id, err)
		return
	}
	defer stream.Close()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	write := func(text string) error {
		data, _ := json.Marshal(strings.TrimRight(text, "\r\n"))
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	tty := inspect.Container.Config != nil && inspect.Container.Config.Tty
	if tty {
		scanner := bufio.NewScanner(stream)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			if write(scanner.Text()) != nil {
				return
			}
		}
		return
	}
	header := make([]byte, 8)
	for {
		if _, err = io.ReadFull(stream, header); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			return
		}
		size := binary.BigEndian.Uint32(header[4:])
		if size == 0 {
			continue
		}
		payload := make([]byte, size)
		if _, err = io.ReadFull(stream, payload); err != nil {
			return
		}
		for _, line := range strings.Split(strings.TrimRight(string(payload), "\r\n"), "\n") {
			if write(line) != nil {
				return
			}
		}
	}
}
