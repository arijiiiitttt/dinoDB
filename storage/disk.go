package storage

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
)

type DiskManager struct {
	mu       sync.RWMutex
	file     *os.File
	filePath string
	numPages uint64
}

func NewDiskManager(filePath string) (*DiskManager, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("open db file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat db file: %w", err)
	}
	numPages := uint64(info.Size()) / PageSize
	return &DiskManager{file: f, filePath: filePath, numPages: numPages}, nil
}

func (dm *DiskManager) WritePage(page *Page) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	offset := int64(page.ID) * PageSize
	_, err := dm.file.WriteAt(page.Data[:], offset)
	if err != nil {
		return fmt.Errorf("write page %d: %w", page.ID, err)
	}
	if page.ID >= dm.numPages {
		dm.numPages = page.ID + 1
	}
	return dm.file.Sync() 
}

func (dm *DiskManager) ReadPage(pageID uint64) (*Page, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if pageID >= dm.numPages {
		return nil, fmt.Errorf("page %d does not exist", pageID)
	}
	page := &Page{ID: pageID}
	offset := int64(pageID) * PageSize
	_, err := dm.file.ReadAt(page.Data[:], offset)
	if err != nil {
		return nil, fmt.Errorf("read page %d: %w", pageID, err)
	}
	return page, nil
}

func (dm *DiskManager) AllocatePage() (*Page, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	page := &Page{ID: dm.numPages}
	dm.numPages++
	offset := int64(page.ID) * PageSize
	_, err := dm.file.WriteAt(page.Data[:], offset)
	if err != nil {
		return nil, fmt.Errorf("allocate page: %w", err)
	}
	return page, nil
}

func (dm *DiskManager) NumPages() uint64 {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.numPages
}

func (dm *DiskManager) Close() error {
	if err := dm.file.Sync(); err != nil {
		return err
	}
	return dm.file.Close()
}

func EncodeUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func DecodeUint64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
