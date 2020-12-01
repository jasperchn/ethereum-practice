package database

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

/**
取memorydb中的部分接口，以map作为存储结构
实现了以下基本接口
	KeyValueReader
	KeyValueWriter
	io.Closer
还使用到了源码中common包的工具方法

------------------------------------------------
这层数据库操作的地位相当于数据库驱动
*/

var (
	errMemorydbClosed = errors.New("database closed")
	errMemorydbNotFound = errors.New("not found")
)

// 只剩下一个map和读写锁了，其它功能先拿掉
type MemoryDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemoryDatabase() *MemoryDatabase {
	return &MemoryDatabase{
		db: make(map[string][]byte),
	}
}

func (db *MemoryDatabase) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db = nil
	return nil
}

func (db *MemoryDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return false, errMemorydbClosed
	}
	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemoryDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return nil, errMemorydbClosed
	}
	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errMemorydbNotFound
}

func (db *MemoryDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return errMemorydbClosed
	}
	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemoryDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return errMemorydbClosed
	}
	delete(db.db, string(key))
	return nil
}