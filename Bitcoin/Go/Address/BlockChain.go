package BlockInfo

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
)

const dbFile = "blockchain4go.db"
const blocksBucket  = "blocks"
const genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

//定义区块链Blockchain结构体
/*
	版本1 区块链包含区块分组
type Blockchain struct {
	Blocks []*Block
}
 */

//版本2 区块链结构体包含指向最后一个区块哈希值和数据库连接
// 通过结合tip和Db就可以对区块链进行操作，包括添加区块、遍历整个区块
type Blockchain struct {
	tip []byte
	Db  *bolt.DB
}

// 判断区块链数据库是否存在
func dbExists() bool  {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

//通过创建创世纪区块，从而创建一条区块链
/* 版本1，直接将区块链下的区块加载进内存，从而生成一条区块链实例
func NewBlockChain() *Blockchain  {
	genesisBlock := NewGenesisBlock()
	return &Blockchain{[]*Block{genesisBlock}}
}
*/

/*
	版本2 ，通过生成一个包含指向最后一个区块的哈希、数据库连接的区块链实例

    1、打开一个数据库文件
    2、检查文件里面是否已经存储了一个区块链
    3、如果已经存储了一个区块链：
        创建一个新的 Blockchain 实例
        设置 Blockchain 实例的 tip 为数据库中存储的最后一个块的哈希
    4、如果没有区块链：
        创建创世块
        存储到数据库
        将创世块哈希保存为最后一个块的哈希
        创建一个新的 Blockchain 实例，初始时 tip 指向创世块（tip 有尾部，尖端的意思，在这里 tip 存储的是最后一个块的哈希）

func NewBlockChain() *Blockchain {
	var tip []byte

	//打开一个BoltDB文件
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	//创建一个读写事务，结尾通过返回nil来提交事务，也可以在结尾之前任意点返回错误来回滚事务
	err = db.Update(func(tx *bolt.Tx) error {
		//获取指定存储区块为blocks的bucket
		b := tx.Bucket([]byte(blocksBucket))

		//如果不存在，则开始生成创世块，创建名称为blocks的bucket，
		// 并将生成的创世块序列化，以key-value进行保存，和将key=l指定为创世块的哈希
		// 如果存在，则获取指向key=l的最后区块哈希值
		if b == nil {
			fmt.Println("No existing blockchain found. Creating a new one...")
			genesis := NewGenesisBlock()
			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				log.Panic(err)
			}

			//以key为区块哈希，value为区块序列化进行区块保存
			err = b.Put(genesis.Hash, genesis.Serialize())
			if err != nil {
				log.Panic(err)
			}

			//更新key=l 指向数据库的最后区块的哈希
			err = b.Put([]byte("l"), genesis.Hash)
			if err != nil {
				log.Panic(err)
			}
			//将tip指向数据库的最后区块的哈希
			tip = genesis.Hash
		} else {
			tip = b.Get([]byte("l"))
		}

		//对当前的数据库读写事务进行提交
		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	// 通过tip(指向数据库中存储的最后区块的哈希值)、数据库连接生成区块链实例
	bc := Blockchain{tip, db}

	return &bc
}
*/

// 从当前数据库dbFile-blockchain4go.db构建区块链实例（指向最后一个区块的tip和数据库连接db）
func GetBlockchain4db() *Blockchain {
	if dbExists() == false {
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}
	return &bc
}

/*
	创建一个只包含创世区块的区块链实例，并生成区块链对应的数据库
	并将创世纪块的奖励给地址address
*/
func CreateBlockchain(address string) *Blockchain  {
	if dbExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
		genesis := NewGenesisBlock(cbtx)

		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}

		tip = genesis.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

// 返回当前区块链实例下地址对应的公钥哈希下所有的未花费的交易
// 1、遍历区块链数据库下的所有区块
// 2、从每个区块中遍历每笔交易
// 3、对每笔交易的输出进行判断，看公钥哈希是否对应其锁定脚本，
//  同时找出每笔交易的输入是否有地址公钥哈希的解锁脚本，若有，则收集对应的交易Id和输出索引号，即已花费的输出
//  最后若每笔输出不在已花费的输出中，则当前交易的输出存在未花费的输出
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for  {
		// 从区块链实例中遍历区块
		block := bci.Next()

		// 对区块的每笔交易进行遍历
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			// 对交易的每笔输出进行遍历
			for outIndex, out := range tx.Vout {

				//判断当前交易ID是否在已花费的输出中，同时索引号也符合条件，则跳出当前循环，继续下一个循环
				if spentTXOs[txID] != nil {
					for _, spentTXOs := range spentTXOs[txID] {
						if spentTXOs == outIndex {
							continue Outputs
						}
					}
				}

				// 判断当前交易输出 是否有公钥哈希，若是，则属于未花费的输出
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}

			}

			//判断当前交易是否是Coinbase交易，只有不是Coinbase交易才会有输入
			// 才能收集所有已花费的交易，即交易Id及对应的输出索引号
			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.VoutIndex)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return unspentTXs
}

// 获取当前区块链中地址对应的公钥哈希下所有的UTXO，即返回交易输出TXOutput数组，可用于计算当前地址的余额
// 1、获取地址对应的公钥哈希下区块链所有未花费的交易
// 2、遍历所有未花费的交易，收集所有的未花费输出
func (bc *Blockchain) FindUTXO(pubKeyHash []byte) []TXOutput  {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

// 从区块链实例中找到当前地址下公钥哈希中数量为amount的可花费的输出，
// 返回可花费的币数（ >amount ）和 交易ID与输出索引号数组的映射，用于转账时构建交易输入
func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIndex, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIndex)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

//增加内容为data的区块到区块链中
/*
	版本1：通过将生成的区块以数组的形式加进区块链中
func (bc *Blockchain) AddBlock(data string)  {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
}
 */

/*
	版本2：根据data内容生成区块，并存储进数据库，并更新区块链实例

func (bc *Blockchain) AddBlock(data string)  {
	var lastHash []byte

	//创建一个只读事务，从数据库中获取指向最后区块的哈希
	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	//通过区块数据+上一个区块哈希来生成一个新的区块
	newBlock := NewBlock(data, lastHash)

	//创建一个读写事务，将生成的区块保存进数据库，并更key=l的内容
	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}
		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		//将区块链实例的tip变量进行更新，指向数据库中的最后一个区块的哈希
		bc.tip = newBlock.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}
*/


/*
	版本3：根据交易生成区块，并存储进数据库，并更新区块链实例
	1、从区块链实例中的数据库连接获取最后一个区块的哈希
	2、根据最后一个区块哈希和交易进行区块生成
	3、成功生成后，将新生成的区块关联到区块的最后
 */
func (bc *Blockchain) MineBlock(transaction []*Transaction)  {
	var lastHash []byte
	
	for _, tx := range transaction {
		if bc.VerifyTransaction(tx) != true {
			log.Panic("Error：Invalid transaction")
		}
	}

	//创建一个只读事务，从数据库中获取指向最后区块的哈希
	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	//通过区块数据+上一个区块哈希来生成一个新的区块
	newBlock := NewBlock(transaction, lastHash)

	//创建一个读写事务，将生成的区块保存进数据库，并更key=l的内容
	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}
		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		//将区块链实例的tip变量进行更新，指向数据库中的最后一个区块的哈希
		bc.tip = newBlock.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}

/*
	对交易通过私钥进行签名
	获取交易输入所引用的交易id-Transaction映射，结合私钥对交易进行签名
 */
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey)  {
	prevTxs :=  make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	tx.Sign(privKey, prevTxs)
}

/*
	对交易进行验证
	1、获取当前交易中的每个输入在区块链引用的交易
	2、对交易及交易输入引用的交易进行签名验证
 */
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool  {
	prevTXs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)]= prevTX
	}

	return tx.Verify(prevTXs)
}

/*
	在当前区块链实例中遍历，查找交易ID对应的交易
	1、对当前区块链实例中的区块进行遍历
	2、将每个区块中的每笔交易的ID与参数ID进行比较
 */
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()
		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found")
}

//创建一个区块链迭代器结构体，包含当前区块哈希和数据库连接
type BlockchainIterator struct {
	currentHash []byte
	db *bolt.DB
}

//返回区块链实例对应的迭代器
func (bc *Blockchain)Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.Db}
	return bci
}

//通过区块链迭代器来返回对应的区块数据，然后指向上一个区块哈希
func (i *BlockchainIterator)Next() *Block  {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}