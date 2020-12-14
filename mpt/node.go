package mpt

import (
	"bytes"
	"ethereum-practice/rlp"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"io"
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
	valueNode	[]byte
)

// 找出公共前缀长度
func commonKeyLength(a []byte, b[]byte) int {
	var minLength = len(a)
	if minLength > len(b) { minLength = len(b)}

	i := 0
	for ;; {
		if i < minLength {
			if a[i] != b[i] { break }
			i++
		} else {
			break
		}
	}
	return i
}


// 尝试解析hashedNode
// 对应源码中的func (t *Trie) resolveHash

// prefix用来打印路径，不是特别重要
func resolveHash(db *Database, hash common.Hash, prefix []byte) (node, error) {
	encoded, err := db.diskdb.Get(hash[:])
	if err != nil || encoded == nil {return nil, err}

	n, err := decodeNode(hash[:], encoded)
	if err != nil { panic(fmt.Sprintf("node %x: %v", hash, err)) }
	return n, err
}

func resolveHashedNode(db *Database, node hashedNode, prefix []byte) (node, error) {
	// 32位截断，和序列化时一致
	return resolveHash(db, common.BytesToHash(node), prefix)
}

/**
编码逻辑
*/
// 注意branch的
func (n *branchNode) EncodeRLP(w io.Writer) error {
	var nodes [17]node
	for i, child := range &n.Children {
		if nil != child {
			nodes[i] = child
		} else { // 悬空的节点要填上可以序列化的值
			nodes[i] = valueNode(nil)
		}
	}
	return rlp.Encode(w, nodes)
}

// shortNode本身只暴露Key和Value，核心代码是用反射实现取值
func (n *shortNode) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, n)
}

func (n *shortNode) EqualsKey(hexKey []byte, startsFrom int) bool {
	return bytes.Equal(n.Key, hexKey[startsFrom : startsFrom + len(n.Key)])
}

//func (n *branchNode) GetChild(b byte) node {
//	if b >= 16 {
//		panic("branch node has only 16 children")
//	}
//	return n.Children[b]
//}
//
//func (n *branchNode) GetValue() node {
//	return n.Children[16]
//}

func (n *shortNode) copy() *shortNode {copy := *n; return &copy}
func (n *branchNode) copy() *branchNode {copy := *n; return &copy}

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
	if count, _ := rlp.CountValues(elements);
	count == 2 { 			 	// rlp编码有两个编码对象时为shortNode，[RLP(Key), RLP(Value)]
		hpeKey, restOfKey, err := rlp.SplitString(elements)
		if err != nil { return nil, err }
		hexKey := hpe2hex(hpeKey)
		if isLeaf(hexKey) { 	// 叶子节点，value解码出来直接复制到shortNode.Val即可
			value, _, err := rlp.SplitString(restOfKey)
			if err != nil {return nil, fmt.Errorf("error occurs when parsing valueNode for shortNode")}
			return &shortNode{hexKey, append(valueNode{}, value...), nodeStatus{hash:hash}}, nil
		} else {				// 非叶子节点 ==> 拓展节点，拓展节点可以接很复杂的子树，必然是递归解码
			subTreeRoot, _, err := decodeSubTreeRecursively(restOfKey)
			if err != nil {return nil, fmt.Errorf("error occurs when parsing subTreeNode for shortNode")}
			return &shortNode{hexKey, subTreeRoot, nodeStatus{hash:hash}}, nil
		}
	} else if count == 17 {		// rlp编码有17个编码对象时为branchNode，[RLP(child_0), ... , RLP(child_16)]，解码逻辑和编码逻辑严格对应
		nd := &branchNode{status:nodeStatus{hash:hash}}
		// 前16个child一定是子树的root
		for i := 0; i < count-1; i++ {
			child, restOfChild, err := decodeSubTreeRecursively(elements)
			if err != nil {return nil, fmt.Errorf("error occurs when parsing %d-th child of branchNode", i)}
			nd.Children[i] = child
			elements = restOfChild
		}
		// 第17个节点如果非空一定是valueNode
		value, _, err := rlp.SplitString(elements)
		if err != nil {return nil, fmt.Errorf("error occurs when parsing value node of branchNode")}
		if len(value) > 0 {
			nd.Children[count] = append(valueNode{}, value...)
		}
		return nd, nil
	} else {					// rlp编码模式越界
		return nil, fmt.Errorf("invalid rlp encoded object, the number of encoded objects within a object must be 2 or 17")
	}
}

// 需要注意的点：
// 原则上，每个节点都应当做RLP编码，然后用sha3计算其哈希作为key。但是存在一种情况，当节点占用空间比哈希key（32byte）还要小时，规范的做法就会浪费存储空间。
// 因此，一般会采取“嵌入节点”（embedded node）的策略，即当节点本身数据大小小于哈希key大小（且不为根节点），则将其直接存储到父节点当中
// 具体地，在解码规则中，
// 1.String类型必然对应hashedNode，它一定做过sha3，所以其大小只能是0或32
// 2.List类型中的元素，可能是hashedNode，也可能是上述的嵌入节点，其合法范围可能是1~32
// 3.上述的“节点本身数据占用空间”是指rlp序列化的bytes，包含了rlp头部的长度标识
func decodeSubTreeRecursively(rlpEncoded []byte) (node, []byte, error) {
	kind, value, restOfValue, err := rlp.Split(rlpEncoded)
	//if err != nil {return nil, rlpEncoded, err}
	if err != nil {return nil, nil, err}

	if rlp.List == kind {
		if decodedBytes := len(rlpEncoded) - len(restOfValue); decodedBytes > common.HashLength {
			return nil, nil, fmt.Errorf("embedded node size must NOT be greater than 32 bytes, which is currently %d bytes", decodedBytes)
		} else { // 合法List类型编码，递归解析
			nd, err := decodeNode(nil, rlpEncoded)
			return nd, restOfValue, err
		}
	} else if rlp.String == kind {    // 实质上的递归终止条件，真的解析到原始字符串/byte数组的数据了
		if len(value) == 0 {
			return nil, restOfValue, nil
		} else if len(value) == common.HashLength {
			return append(hashedNode{}, value...), restOfValue, nil
		} else {
			return nil, nil, fmt.Errorf("RLP String size must be either 0 or 32")
		}
	} else {
		return nil, nil, fmt.Errorf("ecounter invalid RLP type: %s", kind)
	}
}

