package BlockInfo

import (
	"crypto/sha256"
)

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left *MerkleNode
	Right *MerkleNode
	Data []byte
}

/*
	构建MerkleNode
	1、若是叶节点，则left,right为nil，则初始化Data内容为哈希
	2、否则，将新构建MerkleNode的left、right指向参数，data的内容为left、right拼接后的哈希
 */
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}

/*
	计算data [][]byte 的MerkleTree（即根节点），用于计算交易的Merkle根哈希
	1、将data进行偶数初始化，即data的切片长度为奇数，则将最后一个节点进行复制，添加到data尾
	2、将data的内容转换成[]MerkleNode，即将data的内容（交易序列化数据）全部转换成叶节点
	3、对nodes（[]MerkleNode）进行两两组成，生成MerkleNode，最终的节点为根节点
	注意事项：若叶节点的数量不是2的幂次方，在组合过程中，会存在奇数个的节点，
	则最后的节点不用组成，直接参与下一次组合
 */
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, *node)
	}

	for i := 0; i<len(data)/2; i++ {
		var newLevel []MerkleNode
		for j:=0; j<len(nodes); j+=2 {
			if j+1 == len(nodes) {
				newLevel = append(newLevel, nodes[j])
				break
			}
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			newLevel = append(newLevel, *node)
		}
		nodes = newLevel
	}

	mTree := MerkleTree{&nodes[0]}
	return &mTree
}