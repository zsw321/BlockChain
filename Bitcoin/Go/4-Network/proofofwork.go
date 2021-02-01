/*
	定义与工作量证明（proof of work）相关事项细节
 */
package BlockInfo

import (
	"BlockChain4Go/tools"
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

//难度值，表示区块头的哈希值前targetBits必须是0
const targetBits = 20
const maxNonce = math.MaxInt64

/*
	工作量证明结构体
	包含指向的区块，因为每个区块都要进行工作量证明才是有效区块
	目标值，区块头的哈希值必须小于目标值
 */
type ProofOfWork struct {
	block *Block
	target *big.Int
}

//将区块创建一个新的工作量证明
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	pow := &ProofOfWork{b, target}

	return pow
}

//将工作量证明结构进行数据封装，包含PrevBlockHash、Data、Timestamp、targetBits、nonce
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			pow.block.HashTransactions(),
			tools.IntToHex(pow.block.Timestamp),
			tools.IntToHex(int64(targetBits)),
			tools.IntToHex(int64(nonce)),
		},
		[]byte{},
		)

	return data
}

//对当前工作量证明结构体进行计算，寻找有效的nonce，以符合工作量证明的哈希值
func (pow *ProofOfWork) Run() (int, []byte)  {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("Mining a new block ")
	for nonce < maxNonce  {
		data := pow.prepareData(nonce)

		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 {
			fmt.Printf("\r%x", hash)
			break
		} else {
			nonce++
		}
	}

	fmt.Print("\n\n")
	return nonce, hash[:]
}

//验证工作量证明是否有效
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}


