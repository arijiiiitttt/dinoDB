package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"github.com/arijiiiitttt/dinoDB/index"
	"github.com/arijiiiitttt/dinoDB/storage"
)

type DB struct {
	mu      sync.RWMutex
	disk    *storage.DiskManager
	wal     *storage.WAL
	tables  map[string]*Table
	dataDir string
}

type Table struct {
	Name    string
	Columns []string           
	Index   *index.BTree       
	Records map[string]*Record 
	mu      sync.RWMutex
}

func Open(dataDir string) (*DB, error) {
	disk, err := storage.NewDiskManager(dataDir + "/data.db")
	if err != nil {
		return nil, fmt.Errorf("open disk manager: %w", err)
	}
	wal, err := storage.NewWAL(dataDir + "/wal.log")
	if err != nil {
		return nil, fmt.Errorf("open wal: %w", err)
	}
	db := &DB{
		disk:    disk,
		wal:     wal,
		tables:  make(map[string]*Table),
		dataDir: dataDir,
	}
	if err := db.recover(); err != nil {
		return nil, fmt.Errorf("recovery: %w", err)
	}
	return db, nil
}

func (db *DB) recover() error {
	entries, err := db.wal.Replay()
	if err != nil {
		return err
	}
	committed := make(map[uint64]bool)
	for _, e := range entries {
		if e.Operation == "COMMIT" {
			committed[e.TxID] = true
		}
	}
	for _, e := range entries {
		if !committed[e.TxID] {
			continue
		}
		tbl := db.getOrCreateTable(e.TableName)
		switch e.Operation {
		case "INSERT", "UPDATE":
			var r Record
			if err := json.Unmarshal(e.NewData, &r); err == nil {
				tbl.mu.Lock()
				tbl.Records[r.ID] = &r
				tbl.Index.Insert(index.BTreeKey{Key: r.ID})
				tbl.mu.Unlock()
			}
		case "DELETE":
			tbl.mu.Lock()
			delete(tbl.Records, e.RecordID)
			tbl.Index.Delete(e.RecordID)
			tbl.mu.Unlock()
		}
	}
	return nil
}

func (db *DB) getOrCreateTable(name string) *Table {
	db.mu.Lock()
	defer db.mu.Unlock()
	if tbl, ok := db.tables[name]; ok {
		return tbl
	}
	tbl := &Table{
		Name:    name,
		Index:   index.NewBTree(),
		Records: make(map[string]*Record),
	}
	db.tables[name] = tbl
	return tbl
}

func (db *DB) getTable(name string) (*Table, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	tbl, ok := db.tables[name]
	if !ok {
		return nil, fmt.Errorf("table %q does not exist — create it first:\n  CREATE TABLE %s (id, col1, col2)", name, name)
	}
	return tbl, nil
}

func (db *DB) CreateTable(name string, columns []string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, exists := db.tables[name]; exists {
		return fmt.Errorf("table %q already exists", name)
	}
	db.tables[name] = &Table{
		Name:    name,
		Columns: columns,
		Index:   index.NewBTree(),
		Records: make(map[string]*Record),
	}
	return nil
}

func (db *DB) DropTable(name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, exists := db.tables[name]; !exists {
		return fmt.Errorf("table %q does not exist", name)
	}
	delete(db.tables, name)
	return nil
}

func (db *DB) TableExists(name string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	_, exists := db.tables[name]
	return exists
}

func (db *DB) ListTables() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	names := make([]string, 0, len(db.tables))
	for n := range db.tables {
		names = append(names, n)
	}
	return names
}

func (db *DB) Insert(tableName, id string, data map[string]interface{}) error {
	tbl, err := db.getTable(tableName)
	if err != nil {
		return err
	}
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	if existing, exists := tbl.Records[id]; exists && !existing.Deleted {
		return fmt.Errorf("record %q already exists in table %q", id, tableName)
	}
	record := NewRecord(id, data)
	recordBytes, err := record.Serialize()
	if err != nil {
		return fmt.Errorf("serialize: %w", err)
	}
	if err := db.wal.Append(&storage.WALEntry{
		TxID:      newTxID(),
		Operation: "INSERT",
		TableName: tableName,
		RecordID:  id,
		NewData:   recordBytes,
	}); err != nil {
		return fmt.Errorf("wal insert: %w", err)
	}
	tbl.Records[id] = record
	tbl.Index.Insert(index.BTreeKey{Key: id})
	return nil
}

func (db *DB) Get(tableName, id string) (*Record, error) {
	tbl, err := db.getTable(tableName)
	if err != nil {
		return nil, err
	}
	tbl.mu.RLock()
	defer tbl.mu.RUnlock()

	key := tbl.Index.Search(id)
	if key == nil {
		return nil, fmt.Errorf("record %q not found in table %q", id, tableName)
	}
	record, ok := tbl.Records[id]
	if !ok || record.Deleted {
		return nil, fmt.Errorf("record %q not found in table %q", id, tableName)
	}
	return record, nil
}

func (db *DB) GetAll(tableName string) ([]*Record, error) {
	tbl, err := db.getTable(tableName)
	if err != nil {
		return nil, err
	}
	tbl.mu.RLock()
	defer tbl.mu.RUnlock()

	var results []*Record
	for _, r := range tbl.Records {
		if !r.Deleted {
			results = append(results, r)
		}
	}
	return results, nil
}

func (db *DB) Filter(tableName, field, value string) ([]*Record, error) {
	all, err := db.GetAll(tableName)
	if err != nil {
		return nil, err
	}
	var results []*Record
	for _, r := range all {
		if v, ok := r.Data[field]; ok {
			if fmt.Sprintf("%v", v) == value {
				results = append(results, r)
			}
		}
	}
	return results, nil
}

func (db *DB) Search(tableName, query string) ([]*Record, error) {
	all, err := db.GetAll(tableName)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	var results []*Record
	for _, r := range all {
		for _, v := range r.Data {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), query) {
				results = append(results, r)
				break
			}
		}
	}
	return results, nil
}

func (db *DB) Update(tableName, id string, updates map[string]interface{}) (*Record, error) {
	tbl, err := db.getTable(tableName)
	if err != nil {
		return nil, err
	}
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	record, ok := tbl.Records[id]
	if !ok || record.Deleted {
		return nil, fmt.Errorf("record %q not found in table %q", id, tableName)
	}
	oldBytes, _ := record.Serialize()
	for k, v := range updates {
		record.Data[k] = v
	}
	record.UpdatedAt = time.Now()
	newBytes, err := record.Serialize()
	if err != nil {
		return nil, err
	}
	if err := db.wal.Append(&storage.WALEntry{
		TxID:      newTxID(),
		Operation: "UPDATE",
		TableName: tableName,
		RecordID:  id,
		OldData:   oldBytes,
		NewData:   newBytes,
	}); err != nil {
		return nil, fmt.Errorf("wal update: %w", err)
	}
	tbl.Records[id] = record
	return record, nil
}

func (db *DB) Delete(tableName, id string) error {
	tbl, err := db.getTable(tableName)
	if err != nil {
		return err
	}
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	record, ok := tbl.Records[id]
	if !ok || record.Deleted {
		return fmt.Errorf("record %q not found in table %q", id, tableName)
	}
	if err := db.wal.Append(&storage.WALEntry{
		TxID:      newTxID(),
		Operation: "DELETE",
		TableName: tableName,
		RecordID:  id,
	}); err != nil {
		return fmt.Errorf("wal delete: %w", err)
	}
	record.Deleted = true
	record.UpdatedAt = time.Now()
	tbl.Index.Delete(id)
	return nil
}

func (db *DB) HardDelete(tableName, id string) error {
	tbl, err := db.getTable(tableName)
	if err != nil {
		return err
	}
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	if _, ok := tbl.Records[id]; !ok {
		return fmt.Errorf("record %q not found", id)
	}
	delete(tbl.Records, id)
	tbl.Index.Delete(id)
	return nil
}

func (db *DB) Upsert(tableName, id string, data map[string]interface{}) (*Record, error) {
	tbl, err := db.getTable(tableName)
	if err != nil {
		return nil, err
	}
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	if existing, ok := tbl.Records[id]; ok && !existing.Deleted {
		for k, v := range data {
			existing.Data[k] = v
		}
		existing.UpdatedAt = time.Now()
		return existing, nil
	}
	record := NewRecord(id, data)
	tbl.Records[id] = record
	tbl.Index.Insert(index.BTreeKey{Key: id})
	return record, nil
}

func (db *DB) Close() error {
	if err := db.disk.Close(); err != nil {
		return err
	}
	return db.wal.Close()
}

var txCounter uint64
var txMu sync.Mutex

func newTxID() uint64 {
	txMu.Lock()
	defer txMu.Unlock()
	txCounter++
	return txCounter
}
