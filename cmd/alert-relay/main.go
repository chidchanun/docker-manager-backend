package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type payload struct {
	Status string `json:"status"`
	Alerts []struct {
		Status      string            `json:"status"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	} `json:"alerts"`
}

func main() {
	target := os.Getenv("NTFY_URL")
	if target == "" {
		target = "http://ntfy:80/docker-manager-alerts"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("POST /alerts", func(w http.ResponseWriter, r *http.Request) {
		var body payload
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024)).Decode(&body); err != nil {
			http.Error(w, "invalid payload", 400)
			return
		}
		for _, alert := range body.Alerts {
			summary := alert.Annotations["summary"]
			if summary == "" {
				summary = alert.Labels["alertname"]
			}
			description := alert.Annotations["description"]
			message := fmt.Sprintf("Status: %s\nAlert: %s", strings.ToUpper(alert.Status), summary)
			if description != "" {
				message += "\n" + description
			}
			request, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, target, bytes.NewBufferString(message))
			request.Header.Set("Title", "Docker Manager: "+alert.Labels["alertname"])
			request.Header.Set("Tags", "warning,docker")
			if alert.Labels["severity"] == "critical" {
				request.Header.Set("Priority", "urgent")
			} else {
				request.Header.Set("Priority", "high")
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				log.Printf("publish ntfy: %v", err)
				http.Error(w, "notification failed", 502)
				return
			}
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if response.StatusCode >= 300 {
				http.Error(w, "ntfy rejected notification", 502)
				return
			}
		}
		w.WriteHeader(204)
	})
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
