package engine

import (
	"encoding/json"
	"fmt"
	"time"
)

// Record represents one row stored in the database.
type Record struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Deleted   bool                   `json:"deleted"` 
}

// NewRecord creates a new record with timestamps set to now.
func NewRecord(id string, data map[string]interface{}) *Record {
	now := time.Now()
	return &Record{
		ID:        id,
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		Deleted:   false,
	}
}

func (r *Record) Serialize() ([]byte, error) {
	return json.Marshal(r)
}

func Deserialize(data []byte) (*Record, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("deserialize record: %w", err)
	}
	return &r, nil
}
