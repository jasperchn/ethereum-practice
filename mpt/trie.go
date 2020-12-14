package mpt

/*
实现mpt树，对一部分功能做简化处理，先考虑核心逻辑的实现：
	1.database用map替代，不依赖leveldb，实际上就是源码中的memorydb的做法
	2.节点缓存功能暂不实现，主要影响由key查找相应节点的实现——拿掉缓存之后，直接到db中找
	3.rlp编码，暂时先用源码提供的
	4.sha3系列算法暂时使用源码提供的

*/

import (
	"bytes"
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
	Insert
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

//
func (t *Mpt) Insert(key, value []byte) error {
	hexKey := key2hex(key)
	if len(value) != 0 {
		_, root, err := t.insert(t.root, valueNode(value), hexKey, nil)
		if err != nil {return err}
		t.root = root
	}
	return nil
}

// 输入参数
// root:递归查找插入位置的起始节点；value：待插入的数据；key：还未处理的键值；prefix为已处理的key
// 输出参数
// isChanged:该节点是否有变化（与缓存有关，不做缓存其实用不到）；rn：更新后的节点
func (t *Mpt) insert(root, value node, hexKey, prefix []byte) (isChanged bool, rn node, err error) {
	// 递归终止条件，找到合适位置，插入
	if len(hexKey) == 0 {
		if val, ok := root.(valueNode); ok {
			return !bytes.Equal(val, value.(valueNode)), value, nil
		}
		return true, value, nil
	}
	switch nRoot := root.(type) {
	case *branchNode:
		isChanged, rn, err := t.insert(nRoot.Children[hexKey[0]], value, hexKey[1:], append(prefix, hexKey[0]))
		if !isChanged || err != nil {return false, rn, err}
		// 刷新当前节点，子树插入新元素，hash值会变（如果有缓存的话）
		nRoot = nRoot.copy()
		nRoot.status = nodeStatus{dirty: true}
		nRoot.Children[hexKey[0]] = rn
		return true, nRoot, nil
	case *shortNode:
		matchedLength := commonKeyLength(hexKey, nRoot.Key)
		// key完全匹配上
		if matchedLength == len(nRoot.Key) {
			isChanged, rn, err = t.insert(nRoot.Value, value, hexKey[matchedLength:], append(prefix, hexKey[:matchedLength]...))
			if !isChanged || err != nil {
				return false, rn, err
			}
			return true, &shortNode{nRoot.Key, rn, nodeStatus{dirty:true}}, nil
		}
		// key没有完全匹配上，在matchedLength分开，此时需要把shortNode分裂成二叉树，数据结构变成branchNode
		var err error
		branch := &branchNode{status:nodeStatus{dirty:true}}
		// 原节点
		_, branch.Children[nRoot.Key[matchedLength]], err = t.insert(nil, nRoot.Value, nRoot.Key[matchedLength+1:], append(prefix, nRoot.Key[:matchedLength+1]...))
		if err != nil {return false, nil, err}
		// 新节点插入
		_, branch.Children[hexKey[matchedLength]], err = t.insert(nil, value, hexKey[matchedLength+1:], append(prefix, hexKey[:matchedLength+1]...))
		// 用新的branchNode替换掉shortNode；如果key完全不重合，就是一个branchNode，否则需要增加一个拓展节点表示公共部分
		if matchedLength == 0 {return true, branch, nil}
		return true, &shortNode{hexKey[:matchedLength], branch, nodeStatus{dirty:true}}, nil
	case hashedNode:
		decodedNode, err := t.resolveHashedNode(nRoot, prefix)
		if err != nil {return false, nil, err}
		isChanged, rn, err := t.insert(decodedNode, value, hexKey, prefix)
		if !isChanged || err != nil {
			return false, rn, err
		}
		return true, rn, nil
	case nil:
		return true, &shortNode{hexKey, value, nodeStatus{dirty:true}}, nil
	default:
		panic(fmt.Sprintf("errors occurs when processing node: %v", root))
	}
}

// 删除
func (t *Mpt) Delete(key []byte) error {
	return nil
}


func concat(s1, s2 []byte) []byte {
	re := make([]byte, len(s1) + len(s2))
	copy(re, s1)
	copy(re[len(s1):], s2)
	return re
}


// 输入参数
// root递归起始的根节点，prefix已处理过的键值，hexKey还未处理的键值
// 输出参数
// isChanged表示树是否有变动，rn为新的根节点
func (t *Mpt) delete(root node, prefix, hexKey []byte) (isChanged bool, rn node, err error) {
	switch nRoot := root.(type) {
	case *branchNode:
		isChanged, rn, err := t.delete(nRoot.Children[hexKey[0]], append(prefix, hexKey[0]), hexKey[1:])
		// 未修改/出错
		if !isChanged || err != nil {
			return false, rn, err
		}
		// 成功修改，更新当前根节点
		nRoot = nRoot.copy()
		nRoot.status = nodeStatus{dirty: true}
		nRoot.Children[hexKey[0]] = rn
		// branchNode理论上是16叉树，如果删除把子节点干掉只剩一个，就需要调整树的结构了
		// 首先要确定到底有几个子节点，如果只有一个，它的位置又是多少
		// -10：16个节点全满；-2：有2个及以上的非空节点；正数[0,15]：仅剩一个非空节点；正数16：16个节点都是空的，但value非空
		loc := -10
		for i, child := range &nRoot.Children {
			if nil != child {
				if loc == -10 {
					loc = i
				} else {
					loc = -2
				}
			}
		}

		if loc >= 0 && loc < 16 { // 删除后只剩下一个节点，需要调整结构
			// 如果子节点是hashedNode还需要到数据库中读取
			var childNode node
			if hashedRoot, ok := nRoot.Children[loc].(hashedNode); ok {
				cn, err := t.resolveHashedNode(hashedRoot, prefix)
				if err != nil {
					return false, nil, err
				}
				childNode = cn
			} else {
				childNode = nRoot.Children[loc]
			}

			// 如果子节点是shortNode，相当于把子节点向上提一层
			if childNode, ok := childNode.(*shortNode); ok {
				newKey := append([]byte{byte(loc)}, childNode.Key...)
				return true, &shortNode{newKey, childNode.Value, nodeStatus{dirty: true}}, nil
			} else {           // 如果子节点是其他类型，todo check this branch
				return true, &shortNode{[]byte{byte(loc)}, nRoot.Children[loc], nodeStatus{dirty: true}}, nil
			}

		} else if loc == 16 { // 变成叶子节点
			return true, &shortNode{[]byte{byte(loc)}, nRoot.Children[loc], nodeStatus{dirty: true}}, nil
		} else { // 2个及以上，保留原结构
			return true, nRoot, nil
		}
	case *shortNode:
		matchedLength := commonKeyLength(hexKey, nRoot.Key)
		if matchedLength < len(nRoot.Key) {
			return false, nRoot, nil
		}
		if matchedLength == len(hexKey) {
			return true, nil, nil
		}
		// 以value为根节点删除，rn是删除动作处理完之后的根节点，它要替代value
		isChanged, rn, err := t.delete(nRoot.Value, append(prefix, hexKey[:len(nRoot.Key)]...), hexKey[len(nRoot.Key):])
		if !isChanged || err != nil {
			return false, nRoot, err
		}
		// 根据删除后子节点的类型决定如何调整树结构
		switch rn := rn.(type) {
		// 向上收缩
		case *shortNode:
			return true, &shortNode{concat(nRoot.Key, rn.Key), rn.Value, nodeStatus{dirty: true}}, nil
		default:
			return true, &shortNode{nRoot.Key, rn, nodeStatus{dirty:true}}, nil
		}
	case valueNode:
		return true, nil, nil
	case hashedNode:
		currentNode, err := t.resolveHashedNode(nRoot, prefix)
		if err != nil {return false, nil, err}
		isChanged, rn, err := t.delete(currentNode, prefix, hexKey)
		if !isChanged || err != nil {return false, rn, err}
		return true, rn, nil
	case nil:
		return false, nil, nil
	default:
		panic(fmt.Sprintf("errors occurs when processing node: %v", root))
	}
}


