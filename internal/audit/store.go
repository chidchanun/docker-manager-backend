package audit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const stream = "docker-manager:audit"

type Entry struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Container string    `json:"container"`
	IP        string    `json:"ip"`
	Status    int       `json:"status"`
	Success   bool      `json:"success"`
	Time      time.Time `json:"time"`
}
type Store struct{ client *redis.Client }

func New(address string) *Store {
	if address == "" {
		return &Store{}
	}
	return &Store{client: redis.NewClient(&redis.Options{Addr: address})}
}
func (s *Store) Add(ctx context.Context, entry Entry) error {
	if s.client == nil {
		return nil
	}
	return s.client.XAdd(ctx, &redis.XAddArgs{Stream: stream, MaxLen: 10000, Approx: true, Values: map[string]any{"user": entry.User, "action": entry.Action, "container": entry.Container, "ip": entry.IP, "status": entry.Status, "success": entry.Success, "time": entry.Time.Format(time.RFC3339Nano)}}).Err()
}
func (s *Store) List(ctx context.Context, count int64) ([]Entry, error) {
	if s.client == nil {
		return []Entry{}, nil
	}
	rows, err := s.client.XRevRangeN(ctx, stream, "+", "-", count).Result()
	if err != nil {
		return nil, err
	}
	result := make([]Entry, 0, len(rows))
	for _, row := range rows {
		values := row.Values
		parsed, _ := time.Parse(time.RFC3339Nano, stringValue(values["time"]))
		status, _ := values["status"].(int64)
		result = append(result, Entry{ID: row.ID, User: stringValue(values["user"]), Action: stringValue(values["action"]), Container: stringValue(values["container"]), IP: stringValue(values["ip"]), Status: int(status), Success: stringValue(values["success"]) == "1", Time: parsed})
	}
	return result, nil
}
func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
func (s *Store) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
