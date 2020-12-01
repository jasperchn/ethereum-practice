package mpt

/*
实现mpt树，对一部分功能做简化处理，先考虑核心逻辑的实现：
	1.database用map替代，不依赖leveldb，实际上就是源码中的memorydb的做法
	2.节点缓存功能暂不实现，主要影响由key查找相应节点的实现——拿掉缓存之后，直接到db中找
	3.rlp编码，暂时先用源码提供的
	4.sha3系列算法暂时使用源码提供的

*/

import (
	"ethereum-practice/mpt/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// common.Hash 其实就是[32]byte
	EmptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	EmptyState = crypto.Keccak256Hash(nil)
)

type Mpt struct {
	db *database.MemoryDatabase
	root node
}

func New(root common.Hash, db *database.MemoryDatabase) (*Mpt, error){
	if db == nil {
		panic("database can not be nil")
	}

	mpt := &Mpt{db:db}
	if root != (common.Hash{}) && root != EmptyRoot {

	}


}

// 根据key值（已sha3）找出相应的node
func (t *Mpt) resolveHashedNode(n ){}