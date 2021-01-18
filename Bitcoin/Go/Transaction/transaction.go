package BlockInfo

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

const subsidy  = 10

//交易结构体，包括
// 交易ID -- 将交易的输入和输出统一序列化后进行哈希
// Vin — 交易输入数组
// Vout — 交易输出数组
type Transaction struct {
	ID 		[]byte
	Vin 	[]TXInput
	Vout  	[]TXOutput
}

// 交易输入结构体
// 交易Id，指向某笔交易，代表解锁该笔交易的输出
// 交易索引号，指向某笔交易输出的索引号，结合交易Id，用于指向某笔交易的某项输出
// ScriptSig，解锁签名脚本，这里代表花费的地址
type TXInput struct {
	Txid		[]byte
	VoutIndex	int
	ScriptSig	string
}

// 交易输出结构体
// Value：花费的币数，代表给某个地址发送的币数
// ScriptPubKey，锁定脚本，目前代表某个地址
type TXOutput struct {
	Value 			int
	ScriptPubKey	string
}

// 用于设置交易的ID值，
// 在构建交易过程中，先填充输入、输出结构体数组，然后序列化，
// 最后进行哈希，结果即交易ID值
func (tx *Transaction) SetID()  {
	var encoded bytes.Buffer
	var hash [32]byte

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// 判断交易的某笔输入是否可以由unlockingData进行解锁，从而确定币已经被花费
func (in *TXInput) CanUnlockOutputWith(unlockingData string) bool {
	return in.ScriptSig == unlockingData
}

// 判断交易的某笔输出是否可以由unlockingData锁定，从而确定UTXO
func (out *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
	return out.ScriptPubKey == unlockingData
}

// 生成Coinbase交易，用于在挖区块过程，给矿工的奖励
func NewCoinbaseTX(to, data string) *Transaction  {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}

	txin := TXInput{[]byte{}, -1, data}
	txout := TXOutput{subsidy, to}
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{txout}}
	tx.SetID()

	return &tx
}

// 判断某笔交易是否是Coinbase交易
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].VoutIndex == -1
}

// 从from给to发送amount币时，构建一笔交易
// 1、从from中寻找amount数量（至少）的交易输出（即UTXO）
// 2、若成功找到，则开始构建交易输入、交易输出，生成一笔交易
func NewTransaction(from, to string, amount int, bc *Blockchain) *Transaction  {
	var inputs 	[]TXInput
	var outputs	[]TXOutput

	acc, validOutputs := bc.FindSpendableOutputs(from, amount)

	if acc < amount {
		log.Panic("ERROR：Not enough funds")
	}

	for txid, outs := range validOutputs {
		txIDStr, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, outIndex := range outs {
			input := TXInput{txIDStr, outIndex, from}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TXOutput{amount, to})
	if acc > amount {
		outputs = append(outputs, TXOutput{acc-amount, from})
	}

	tx := Transaction{nil, inputs, outputs}
	tx.SetID()

	return &tx
}