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
		==> 叶子节点和拓展节点的数据结构是一样的，可以通过key的格式来区分。数据结构为shortNode
	（2）分支节点本质上是16叉数+1个value
		===> 结构特殊，它应该有自己的数据结构。数据结构为branchNode
	（3）对于数据库操作，应允许由已哈希的key查找节点，对于任一节点，在数据库中还有一种等价的表示形式
		===> []byte类型，用于表示已哈希的节点（key值）。数据结构为hashNode，实质为[]byte

关于解码的实现：
	序列化串反序列化为节点对象可以理解为shortNode, branchNode的一种构造方法，从此角度，其实现可放到此类；
	另外，由于rlp的递归特点，其反序列化也应当放到rlp包外部、节点的定义类中。
	原因如下：
	rlp的编码本身是递归的，它可以编码：（a）字符串/byte数组（b）列表（c）可编码对象构成的列表
	特别地，对于struct类型的数据，需要明确将结构体压平为list的规则，只有确定了编码规则才能实现解码；
	在rlp包中，首先encode.go提供了makeWriter()，它根据待编码对象的类型做分发；然后，针对struct分发至makeStructWriter()；
	最后，用反射依次取出第一个属性到最后一个公有属性，递归地将它们处理成列表形式
	综上应当注意：
	1.struct中属性的顺序与访问权限会影响到结构体压平，进而影响rlp编码，自然也会影响rlp解码
	2.严格地说，rlp编码本身不涉及如何把结构体拉平（这算是对rlp的合理拓展），目前的规则只是一种合理的规则，完全可以改变。
	按编解码对应的原则，本应当在rlp包中有相应的解码实现，但是由于rlp的递归特点，它的可编码对象有无穷多种，根本无法对应起来。
	所以，编码实现放在rlp包中，而解码逻辑一定是放在rlp包之外、针对特定的数据结构（shortNode和branchNode）实现的

	主要会用到 Split SplitList SplitString 函数，返回值都是获得第一个Rlp串（Rlp开头是长度标记）和剩余部分，非常适合递归算法
	CountValues返回编码中存在的对象数量

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