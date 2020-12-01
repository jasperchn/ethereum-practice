package mpt

/*
以太坊中的节点最终会转化成<key, value>保存，先考虑key的编码
key有三种编码形式：
	1 原始key：[]byte类型，纯原生
	2 Hex encoding：[]byte类型，每个字节存储一个半字节(nibble)，末尾字节作为判断符，表示是否是叶子节点。
		2.1 由于SHA3得到的输出是16进制字符串，存储它实质上只需要用低4位，用8位是空间的浪费
		2.2 拓展节点和叶子节点在数据结构上完全一致，需要使用判断位，若末尾byte为0b 0001 0000(0x10)则为叶子节点，否则为拓展节点
			如果纯粹作为标识符，[16, 256]范围的数都可以
	3 Hex Prefix encoding：[]byte类型，每个字节存储两个半字节，第一个字节包含了 (a) nibble奇偶性 （b）节点是否为叶子节点 信息
		3.1 HPE的第一个byte有4种可能：
			0011 xxxx	节点为叶子节点，nibble奇数个，xxxx为第一个nibble
			0010 0000	节点为叶子节点，nibble偶数个
			0001 xxxx	节点为拓展节点，nibble奇数个，xxxx为第一个nibble
			0000 0000	节点为拓展节点，nibble偶数个
			总之，0b 0010 0000（0x20）取出叶子/拓展节点判断位； 0b 0001 0000 (0x10) 取出nibble奇偶位
		3.2 HPE第二个byte开始，是两个半字节的数据“挤”到一个字节的空间中

此类提供3中编码方式的相互转换，对应源码的trie/encoding.go

原始key					KEYBYTES		key
Hex encoding			HEX				hex
Hex Prefix encoding		COMPACT			hpe

*/

const (
	HexLeafFlag       = 0x10 // 原则上取[16,256]都可
	HpeOddNibblesFlag = 0x10 // 高四位取1bit作为flag
	HpeLeafFlag       = 0x20 // 高四位取1bit作为flag

	HpeOddNibblesMask = HpeOddNibblesFlag >> 4
	HpeLeafMask       = HpeLeafFlag >> 4
)


// key -> Hex
// 此处有个trick，无论该节点是否是叶子节点，都把判断位加上，由调用方决定取[]还是[:len-1]
func key2hex(key []byte) []byte {
	if len(key) < 0 {
		panic("invalid key input, length < 0")
	}
	hex := make([]byte, len(key) * 2 + 1)
	for i, byte := range key {
		// 高4位、低4为拆到两个byte中，占用空间翻倍
		hex[i*2] = extractHighNibble(byte)
		hex[i*2+1] = extractLowNibble(byte)
	}
	hex[len(hex)-1] = HexLeafFlag
	return hex
}

// 需要考虑到key是否是叶子节点
func key2hpe(key []byte) []byte{
	panic("not implemented")
}

func hex2key(hex []byte) []byte{
	if isLeaf(hex) {
		return nibblesIntoByte(hex[:len(hex) - 1])
	} else {
		return nibblesIntoByte(hex)
	}
}

// 2个条件组合出4种情况，分类讨论
// todo 与源码实现有较大差别，需要对4分支都提供测试用例覆盖
func hex2hpe(hex []byte) []byte{
	boolLeaf, boolOdd := isLeaf(hex), len(hex)&1 == 1
	if boolOdd {
		if boolLeaf{
			hpe := make([]byte, (len(hex)-1)/2+1)
			hpe[0] = HpeOddNibblesFlag | HpeLeafFlag | extractHighNibble(hex[0])
			nibblesIntoByteInplace(hex[:len(hex)-1], hpe[1:])
			return hpe
		} else {
			hpe := make([]byte, len(hex)/2+1)
			hpe[0] = HpeOddNibblesFlag | extractHighNibble(hex[0])
			nibblesIntoByteInplace(hex[:len(hex)-1], hpe[1:])
			return hpe
		}
	} else {
		if boolLeaf{
			hpe := make([]byte, (len(hex)-1)/2+1)
			hpe[0] = HpeLeafFlag
			nibblesIntoByteInplace(hex, hpe[1:])
			return hpe
		} else {
			hpe := make([]byte, len(hex)/2+1)
			nibblesIntoByteInplace(hex, hpe[1:])
			// hpe[0] 自动初始化为0值
			return hpe
		}
	}
}

func hex2hpeInplace(hex []byte){
	panic("not Implemented")
}

// 把hpe从两两nibble占用一个byte反解回一个nibble占用一个byte
// 奇偶、叶子/拓展信息全都在第1个byte里，第2个byte根据奇偶性可能是数据
func hpe2hex(hpe []byte) []byte {
	if len(hpe) == 0 {
		return hpe
	}
	native := key2hex(hpe)

	// 检查是否需要保留叶子/拓展判断符
	if HpeLeafMask & native[0] != HpeLeafMask {
		native = native[:len(native)-1]
	}
	// 检查nibble数是否奇数（native[1]是否作为数据）
	if HpeOddNibblesMask & native[0] == HpeOddNibblesMask {
		return native[1:]
	} else {
		return native[2:]
	}
}


func isLeaf(s []byte) bool {
	return len(s) > 0 && s[len(s) - 1] == HexLeafFlag
}

// 半字节两两压入byte中, inplace
func nibblesIntoByteInplace(nibbles []byte, bytes []byte){
	// nibble的偶数在此处检查
	if len(nibbles) & 1 == 1 {
		panic("length of nibbles must be odd when compressing it into bytes")
	}
	for in, ib := 0, 0; in < len(nibbles); in, ib = in + 2, ib + 1 {
		bytes[ib] = nibbles[in] << 4 | nibbles[in + 1]
	}
}

// 半字节两两压入byte中, 新创建空间
func nibblesIntoByte(nibbles []byte) []byte{
	bytes := make([]byte, len(nibbles) / 2)
	nibblesIntoByteInplace(nibbles, bytes)
	return bytes
}

func extractHighNibble(b byte) byte{return b >> 4}

func extractLowNibble(b byte) byte{return b & 0x0F}