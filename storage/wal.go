package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type WALEntry struct {
	LSN       uint64          `json:"lsn"`              
	TxID      uint64          `json:"tx_id"`            
	Operation string          `json:"op"`               
	TableName string          `json:"table"`
	RecordID  string          `json:"record_id"`
	OldData   json.RawMessage `json:"old_data,omitempty"` 
	NewData   json.RawMessage `json:"new_data,omitempty"` 
	Timestamp time.Time       `json:"ts"`
}

type WAL struct {
	mu      sync.Mutex
	file    *os.File
	writer  *bufio.Writer
	nextLSN uint64
}

func NewWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open wal: %w", err)
	}
	return &WAL{file: f, writer: bufio.NewWriter(f)}, nil
}

func (w *WAL) Append(entry *WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry.LSN = w.nextLSN
	entry.Timestamp = time.Now()
	w.nextLSN++

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal wal entry: %w", err)
	}
	_, err = fmt.Fprintf(w.writer, "%s\n", data)
	if err != nil {
		return err
	}
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Sync() 
}

func (w *WAL) Replay() ([]*WALEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, 0); err != nil {
		return nil, err
	}
	var entries []*WALEntry
	scanner := bufio.NewScanner(w.file)
	for scanner.Scan() {
		var e WALEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue 
		}
		entries = append(entries, &e)
	}
	return entries, scanner.Err()
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Close()
}
