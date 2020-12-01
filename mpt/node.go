package mpt

import (
	"ethereum-practice/rlp"
	"fmt"
)

/**
节点类型与相关方法
以太坊中节点类型：
	空节点，叶子节点，拓展节点，分支节点
	注意：
	（1）叶子节点和拓展节点都是key，value形式，差别是拓展节点的value是其他节点的hash，它有子节点
		==> 叶子节点和拓展节点的数据结构是一样的，可以通过key的格式来区分
	（2）分支节点本质上是16叉数+1个value
		===> 结构特殊，它应该有自己的数据结构
	（3）对于数据库操作，应允许由已哈希的key查找节点，对于任一节点，在数据库中还有一种等价的表示形式
		===> []byte类型，用于表示已哈希的节点（key值）

func:
	1.
	由序列化串转为节点属于node的一种构造方法，实现放到此类中;
	以太坊的序列化/反序列化是rlp编码，实现起来有一定工作量，此处先直接用
	关于rlp：
		主要会用到 Split SplitList SplitString 函数，返回值都是获得第一个Rlp串（Rlp开头是长度标记）和剩余部分，非常适合递归算法
		CountValues返回编码中存在的对象，需要一次遍历

*/

// 节点应当满足的一些公有方法
type node interface {

}

// nodeStatus对应源码的nodeFlag，主要与缓存管理有关
// 特别是注意dirty，他表示内存中的节点与数据库不一致
type nodeStatus struct {
	hash 	hashedNode
	dirty 	bool
}

type(
	branchNode struct {
		Children 	[17]node
		status		nodeStatus
	}
	shortNode struct {
		Key 	[]byte
		Value 	node
		status	nodeStatus
	}
	hashedNode	[]byte
	//valueNode	[]byte
)


func commonKeyLength(a []byte, b[]byte) {}



// 尝试解析hashedNode
// 对应源码中的func (t *Trie) resolveHash
//func resolveHashedNode(db *database.MemoryDatabase, node hashedNode, prefix []byte) (node, error) {
//	// 32位截断，和序列化时一致
//	hash := common.BytesToHash(node)
//	if node := db.
//}



/**
解码逻辑
*/

// hash是节点的哈希值，rlpEncoded是经过rlp编码序列化的串串
// hash本质上和rlp解码没有一点关系，传递过来主要是为了恢复nodeStatus中的hash
// nodeStatus是为缓存保留的，如果纯粹用数据库读写，完全可以把这个参数去掉
func decodeNode(hash, rlpEncoded []byte) (node, error) {
	if len(rlpEncoded) == 0 {return nil, fmt.Errorf("invalid rlpEncoded")}

	elements, _, err := rlp.SplitList(rlpEncoded)
	if err != nil {
		return nil, fmt.Errorf("error occurs during decoding: %v", err)
	}
	//
	if count, _ := rlp.CountValues(elements); count == 2 {

	}

}