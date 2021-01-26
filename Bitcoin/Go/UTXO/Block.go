/*
	实现区块的基本结构
 */

package BlockInfo

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

//定义区块Block结构体
type Block struct {
	Timestamp 	  int64
	Nonce         int
	//Data          []byte
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
}

//计算当前区块的哈希值
/* 通过工作量证明来返回符合目标难度的区块哈希值
func (block *Block) SetHash()  {
	timestamp := []byte(strconv.FormatInt(block.Timestamp, 10))
	//fmt.Println(string(timestamp))
	//fmt.Printf("timestamp：%x\n", timestamp)

	headers := bytes.Join([][]byte{block.PrevBlockHash, block.Data, timestamp}, []byte{})
	//fmt.Println(string(headers))
	fmt.Printf("headers：%x\n", headers)

	hash := sha256.Sum256(headers)

	block.Hash = hash[:]
}
 */

//根据data、前一个区块哈希进行工作量证明，创建一个新的区块
func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block  {
	block := &Block{
		time.Now().Unix(),
		0,
		transactions,
		prevBlockHash,
		[]byte{},
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

//创建创世纪区块
func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

/*
	版本1
	计算一个区块的所有交易的哈希值
	通过将区块的每笔交易的ID进行join后算哈希值
func (b *Block) HashTransactions() []byte  {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}

	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}
*/
func (b *Block) HashTransactions() []byte {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}

	mTree := NewMerkleTree(transactions)

	return mTree.RootNode.Data
}

//将Block区块结构序列化为[]byte
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

//将[]byte内容进行解序列化，返回对应的Block区块结构
func DeserializeBlock(d []byte) *Block  {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}