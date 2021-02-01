package BlockInfo

import (
	"encoding/hex"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
)

const utxoBucket  = "chainstate"


type UTXOSet struct {
	Blockchain *Blockchain
}

/*
	从UTXO集中查找对应公钥哈希和数量的可花费输出（int, map[string][]int）
	1、读取数据库下的UTXO集
	2、对UTXO集进行遍历，查询并返回符合条件的公钥哈希和大于要求数量amount的可花费输出
 */
func (u UTXOSet) FindSpendableOutputs(pubkeyHash []byte, amount int) (int, map[string][]int)  {
	unspentOutputs := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.Db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIndex, out := range outs.Outputs {
				if out.IsLockedWithKey(pubkeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIndex)
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return accumulated, unspentOutputs
}

/*
	打印UTXO集下的交易信息（交易ID、地址、值）
 */
func (u UTXOSet) PrintUTXO()  {
	db := u.Blockchain.Db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)
			fmt.Printf("--- Transaction %x：\n", k)
			for _, out := range outs.Outputs {
				fmt.Printf("address: %s ", PKHashToAddress(out.PubKeyHash))
				fmt.Printf("value：'%d'\n", out.Value)
			}
		}
		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}

/*
	在UTXO集中查找指定公钥哈希对应的输出集（[]TXOutput）
 */
func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TXOutput  {
	var UTXOs []TXOutput
	db := u.Blockchain.Db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)
			fmt.Printf("--- Transaction %x：\n", k)
			for _, out := range outs.Outputs {
				fmt.Printf("address: %s ", PKHashToAddress(out.PubKeyHash))
				fmt.Printf("value：'%d'\n", out.Value)
				if out.IsLockedWithKey(pubKeyHash) {
					//fmt.Println(fmt.Sprintf("--- Transaction %x：", tx.ID))
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return UTXOs
}

/*
	计算UTXO集中包含的交易数（交易ID对应的key）
 */
func (u UTXOSet) CountTransactions() int {
	db := u.Blockchain.Db
	counter := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return counter
}

/*
	从区块链数据库中读取区块交易，重新生成UTXO集，并更新数据库中
	1、删除后并新建区块链数据库下Bucket为utxoBucket的数据
	2、查找数据库下所有的未花费输出（map[string]TXOutputs  交易Id-输出结构体分组）
	3、将上步查找的结果保存进行数据库
	注意事项：当一个新的区块链被创建以后，就会立刻进行重建索引。目前，
	这是 Reindex 唯一使用的地方，即使这里看起来有点“杀鸡用牛刀”，因为一条链开始的时候，
	只有一个块，里面只有一笔交易，Update 已经被使用了。不过我们在未来可能需要重建索引的机制。
 */
func (u UTXOSet) Reindex()  {
	db := u.Blockchain.Db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_, err = tx.CreateBucket(bucketName)
		if err != nil {
			log.Panic(err)
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	UTXO := u.Blockchain.FindUTXO()
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			if err != nil {
				log.Panic(err)
			}

			err = b.Put(key, outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}

/*
	将区块参数中的交易进行遍历更新数据库中UTXO集
	当挖出一个新块时，应该更新 UTXO 集。更新意味着移除已花费输出，并从新挖出来的交易中加入未花费输出。
	1、获取数据库中为chainstate的Bucket对象
	2、对区块中的交易进行遍历
	3、若非coinbase交易，则将交易的输入所引用交易输出从UTXO集删除
	4、将交易的输出更新至UTXO集（不管是不是coinabase交易）
 */
func (u UTXOSet) Update(block *Block)  {
	db := u.Blockchain.Db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, vin := range tx.Vin {
					updateOuts := TXOutputs{}
					outsBytes := b.Get(vin.Txid)
					outs := DeserializeOutputs(outsBytes)

					for outIndex, out := range outs.Outputs {
						//比较的值感觉有点问题，找个时间好好排查
						if outIndex != vin.VoutIndex {
							updateOuts.Outputs = append(updateOuts.Outputs, out)
						}
					}

					if len(updateOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						err := b.Put(vin.Txid, updateOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}
				}
			}

			newOutputs := TXOutputs{}
			for _, out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			fmt.Printf("Update UTXO for block: %x\n", tx.ID)
			err := b.Put(tx.ID, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}
