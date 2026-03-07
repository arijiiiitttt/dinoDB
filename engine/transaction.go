package engine

import (
	"fmt"
	"sync"
	"time"
)

type TxStatus int

const (
	TxActive    TxStatus = iota
	TxCommitted
	TxAborted
)

type Operation struct {
	Type      string 
	TableName string
	RecordID  string
	OldRecord *Record 
	NewRecord *Record 
}

type Transaction struct {
	ID         uint64
	Status     TxStatus
	Operations []Operation
	StartedAt  time.Time
	mu         sync.Mutex
	db         *DB
}

func (db *DB) Begin() *Transaction {
	return &Transaction{
		ID:        newTxID(),
		Status:    TxActive,
		StartedAt: time.Now(),
		db:        db,
	}
}

func (tx *Transaction) TxInsert(tableName, id string, data map[string]interface{}) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.Status != TxActive {
		return fmt.Errorf("transaction %d is not active", tx.ID)
	}
	record := NewRecord(id, data)
	tx.Operations = append(tx.Operations, Operation{
		Type:      "INSERT",
		TableName: tableName,
		RecordID:  id,
		NewRecord: record,
	})
	return nil
}

func (tx *Transaction) TxUpdate(tableName, id string, updates map[string]interface{}) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.Status != TxActive {
		return fmt.Errorf("transaction %d is not active", tx.ID)
	}

	old, err := tx.db.Get(tableName, id)
	if err != nil {
		return err
	}
	oldCopy := *old

	newData := make(map[string]interface{})
	for k, v := range old.Data {
		newData[k] = v
	}
	for k, v := range updates {
		newData[k] = v
	}
	newRecord := NewRecord(id, newData)

	tx.Operations = append(tx.Operations, Operation{
		Type:      "UPDATE",
		TableName: tableName,
		RecordID:  id,
		OldRecord: &oldCopy,
		NewRecord: newRecord,
	})
	return nil
}

func (tx *Transaction) TxDelete(tableName, id string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.Status != TxActive {
		return fmt.Errorf("transaction %d is not active", tx.ID)
	}

	old, err := tx.db.Get(tableName, id)
	if err != nil {
		return err
	}
	tx.Operations = append(tx.Operations, Operation{
		Type:      "DELETE",
		TableName: tableName,
		RecordID:  id,
		OldRecord: old,
	})
	return nil
}

func (tx *Transaction) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.Status != TxActive {
		return fmt.Errorf("transaction %d is not active", tx.ID)
	}

	for _, op := range tx.Operations {
		var err error
		switch op.Type {
		case "INSERT":
			err = tx.db.Insert(op.TableName, op.RecordID, op.NewRecord.Data)
		case "UPDATE":
			_, err = tx.db.Update(op.TableName, op.RecordID, op.NewRecord.Data)
		case "DELETE":
			err = tx.db.Delete(op.TableName, op.RecordID)
		}
		if err != nil {
			tx.Status = TxAborted
			return fmt.Errorf("commit op %s failed: %w — transaction aborted", op.Type, err)
		}
	}
	tx.Status = TxCommitted
	return nil
}

func (tx *Transaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.Status != TxActive {
		return fmt.Errorf("transaction %d is already %v", tx.ID, tx.Status)
	}

	for i := len(tx.Operations) - 1; i >= 0; i-- {
		op := tx.Operations[i]
		switch op.Type {
		case "INSERT":
			_ = tx.db.HardDelete(op.TableName, op.RecordID)
		case "UPDATE":
			if op.OldRecord != nil {
				_, _ = tx.db.Update(op.TableName, op.RecordID, op.OldRecord.Data)
			}
		case "DELETE":
			if op.OldRecord != nil {
				_ = tx.db.Insert(op.TableName, op.RecordID, op.OldRecord.Data)
			}
		}
	}
	tx.Status = TxAborted
	return nil
}
