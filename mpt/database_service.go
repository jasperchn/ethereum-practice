package mpt

import (
	"ethereum-practice/mpt/database"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"sync"
)

/**
这层数据库操作是在database/database.go定义的基础操作（地位相当于数据库驱动）之上，根据业务需要进一步开发
类似@Repository层或@Service层的地位

“业务”主要就是
（1）存：节点 -> 序列化 -> Put/Save
（2）取：hashedKey -> Get -> 反序列化 -> 节点



*/

type KeyValueReader interface {
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
}

type KeyValueWriter interface {
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

type KeyValueStore interface {
	KeyValueReader
	KeyValueWriter
	io.Closer
}

type Database struct {
	diskdb KeyValueStore // Persistent storage for matured trie nodes
	lock sync.RWMutex
}

func NewDatabase() *Database {
	return &Database{diskdb:database.NewMemoryDatabase()}
}

// 根据hashed key取，无缓存情况下非常直接
func (db *Database) resolveHash(hash common.Hash) node {
	n, err := resolveHash(db, hash, nil)
	if err != nil {
		return nil
	}
	return n

	//encoded, err := db.diskdb.Get(hash[:])
	//if err != nil || encoded == nil {
	//	return nil
	//}
	//
	//n, err := decodeNode(hash[:], encoded)
	//if err != nil {
	//	panic(fmt.Sprintf("node %x: %v", hash, err))
	//}
	//return n
}




