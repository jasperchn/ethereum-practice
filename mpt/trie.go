package mpt

/*
实现mpt树，对一部分功能做简化处理，先考虑核心逻辑的实现：
	1.database用map替代，不依赖leveldb，实际上就是源码中的memorydb的做法
	2.节点缓存功能暂不实现，主要影响由key查找相应节点的实现——拿掉缓存之后，直接到db中找
	3.rlp编码，暂时先用源码提供的
	4.sha3系列算法暂时使用源码提供的

*/

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// common.Hash 其实就是[32]byte
	EmptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	EmptyState = crypto.Keccak256Hash(nil)
)

type Mpt struct {
	db *Database
	root node
}


func New(root common.Hash) (*Mpt, error){
	mpt := &Mpt{db:NewDatabase()}
	// 提供root时从数据库中加载
	if root != (common.Hash{}) && root != EmptyRoot {
		rn, err := mpt.resolveHashedNode(root[:], nil)
		if err != nil {
			return nil, err
		}
		mpt.root = rn
	}
	return mpt, nil
}


func (t *Mpt) resolveHash(hash common.Hash, prefix []byte) (node, error) {
	return resolveHash(t.db, hash, prefix)
}

func (t *Mpt) resolveHashedNode(hashedNode hashedNode, prefix []byte) (node, error) {
	return resolveHashedNode(t.db, hashedNode, prefix)
}



/**
基本操作
Insert:

Get:
	1.GetValue, getValueByHex, 根据键值查找相应的value；
	2.GetNode, getNodeByHex, 根据键值查找相应的node，然后返回rlp编码的整个节点
Update:

Delete

Commit/Hash（遍历，编码，sha3，保存）

*/

func (t *Mpt) GetValue(key []byte) ([]byte, error) {
	value, resolvedNode, resolved, err := t.getValueByHex(t.root, key2hex(key), 0)
	if err == nil && resolved {
		t.root = resolvedNode
	}
	return value, err
}

// 输入参数：
// root：查询的根节点；hexKey：查询的key（hex encoding）；keyBias，当前比较key的起始位置
// 输出：
// value：查获的value值；newRoot：下一次递归查找的新节点；resolved：标记符，如果从hashedNode中解析出来，就用对象替换掉哈希值（嵌入了更新策略，理论上可以减少一次更新大量节点的可能）
func (t *Mpt) getValueByHex(root node, hexKey []byte, keyBias int) (value []byte, newRoot node, resolved bool, err error) {
	//return nil, nil, false, nil
	switch nd := (root).(type) {
	case *shortNode:
		// 无法匹配
		if !nd.EqualsKey(hexKey, keyBias) { return nil, nd, false, nil}
		// 递归查找
		value, resolvedNode, resolved, err := t.getValueByHex(nd.Value, hexKey, keyBias+len(nd.Key))
		if err == nil && resolved {
			nd = nd.copy()         // 防止改掉其他地方的值？
			nd.Value = resolvedNode
		}
		return value, nd, resolved, err
	case *branchNode:
		// hexKey中每一个byte只用了低4位，值范围[0, 15]
		value, resolvedNode, resolved, err := t.getValueByHex(nd.Children[hexKey[keyBias]], hexKey, keyBias+1)
		if err == nil && resolved {
			nd = nd.copy()
			nd.Children[hexKey[keyBias]] = resolvedNode
		}
		return value, nd, resolved, err
	case valueNode:
		// 这是正常的递归退出条件
		return nd, nd, false, nil
	case hashedNode:
		// 从数据库取查找并且解码
		decodedNode, err := t.resolveHashedNode(nd, hexKey[:keyBias])
		if err != nil {
			return nil, nd, true, err
		}
		value, resolvedNode, _, err := t.getValueByHex(decodedNode, hexKey, keyBias)
		return value, resolvedNode, true, err
	case nil:
		return nil, nil, false, nil
	default:
		panic(fmt.Sprintf("errors occurs when processing node: %v", root))
	}
}

func (t *Mpt) GetNode(hpeKey []byte) []byte {
	panic("not implemented")
}

func (t *Mpt) getNodeByHex(root node, hexKey []byte, keyBias int) (n node, newRoot node, resolved int, err error) {
	panic("not implemented")
}
