package BlockInfo

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
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
// Signature，签名
// PubKey，公钥
// Signature + PubKey 就是锁定脚本
type TXInput struct {
	Txid		[]byte
	VoutIndex	int
	Signature   []byte
	PubKey    	[]byte
}

// 交易输出结构体
// Value：花费的币数，代表给某个地址发送的币数
// PubKeyHash，公钥哈希，代表锁定脚本
type TXOutput struct {
	Value 			int
	PubKeyHash		[]byte
}

type TXOutputs struct {
	Outputs 	[]TXOutput
}

func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	 txCopy := *tx
	 txCopy.ID = []byte{}

	 hash = sha256.Sum256(txCopy.Serialize())

	 return hash[:]
}

func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := Ripmd160Hash(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

//将交易输出结构体的接收者公钥哈希通过address进行绑定
func (out *TXOutput) Lock(address []byte)  {
	pubKeyHash := Base58Decode(address)
	pubKeyHash = pubKeyHash[1: len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

/*
	生成Coinbase交易，用于在挖区块过程，给矿工的奖励
	Coinbase交易没有输入，即指向的前一笔输入的交易Id为空、索引号为-1、签名为nil，公钥为数据信息
	Coinbase交易的输出（奖励、接收者公钥哈希）
 */
func NewCoinbaseTX(to, data string) *Transaction  {
	if data == "" {
		//data = fmt.Sprintf("Reward to '%s'", to)
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}
		data = fmt.Sprintf("%x", randData)
	}

	txin := TXInput{[]byte{}, -1, nil,[]byte(data)}
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}}
	tx.ID = tx.Hash()

	return &tx
}

//通过value、address信息构建交易输出结构体TXOutput{Value,PubKeyHash}
func NewTXOutput(value int, address string) *TXOutput  {
	txout := &TXOutput{value, nil}
	txout.Lock([]byte(address))

	return txout
}

// 判断某笔交易是否是Coinbase交易
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].VoutIndex == -1
}

/*
	 构建一笔交易，从from给to发送amount币
	1、创建钱包集对象，并获取地址from下的钱包信息（私钥-公钥）
	2、获取区块链中公钥哈希对应的amount数量的输出集
	3、从未花费输出集中构建交易输入
	4、构建交易输出
	5、对交易进行签名

func NewTransaction(from, to string, amount int, bc *Blockchain) *Transaction  {
	var inputs 	[]TXInput
	var outputs	[]TXOutput

	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}

	wallet := wallets.GetWallet(from)
	pubKeyHash := Ripmd160Hash(wallet.PublicKey)

	acc, validOutputs := bc.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("ERROR：Not enough funds")
	}

	for txid, outs := range validOutputs {
		txIDStr, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, outIndex := range outs {
			input := TXInput{txIDStr, outIndex, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	bc.SignTransaction(&tx, wallet.PrivateKey)

	return &tx
}
*/

func NewUTXOTransaction(wallet *Wallet, to string, amount int, utxoSet *UTXOSet) *Transaction {
	var inputs 	[]TXInput
	var outputs	[]TXOutput

	pubKeyHash := Ripmd160Hash(wallet.PublicKey)
	acc, validOutputs := utxoSet.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("ERROR：Not enough funds")
	}

	for txid, outs := range validOutputs {
		txIDStr, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, outIndex := range outs {
			input := TXInput{txIDStr, outIndex, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	from := fmt.Sprintf("%s", wallet.GetAddress())
	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	utxoSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)

	return &tx
}

/*
	在UTXO集基础上构建一笔从from到to的amount的交易，并返回
	1、创建钱包集对象，并获取地址from下的钱包信息（私钥-公钥）
	2、获取UTXO集中公钥哈希对应的amount数量的输出集
	3、从未花费输出集中构建交易输入
	4、构建交易输出
	5、对交易进行签名

func NewUTXOTransaction(from, to string, amount int, utxoSet *UTXOSet) *Transaction {
	var inputs 	[]TXInput
	var outputs	[]TXOutput

	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}

	wallet := wallets.GetWallet(from)
	pubKeyHash := Ripmd160Hash(wallet.PublicKey)

	acc, validOutputs := utxoSet.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("ERROR：Not enough funds")
	}

	for txid, outs := range validOutputs {
		txIDStr, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, outIndex := range outs {
			input := TXInput{txIDStr, outIndex, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	utxoSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)

	return &tx
}
*/

/*
	通过私钥+交易输入引用的交易id-Transaction映射对交易进行签名
	1、对交易的每笔输入进行遍历，判断其引用的交易是否在交易id-Transaction映射中
	2、获取当前交易的修剪版的交易txCopy（清空了每笔输入的签名和公钥哈希）
	3、对交易txCopy的每个输入遍历进行签名
		3.1、先将当前需要签名的输入中的公钥置换为引用的交易输出的公钥哈希
		3.2、交易txCopy的ID赋值为交易txCopy的哈希
		3.3、将当前需要签名的输入中的公钥置为nil
		3.4、对交易txCopy的ID（哈希）通过私钥生成签名
		3.5、将生成的签名赋值到源交易中对应的输入下的Signature字段
 */
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction)  {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for index, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]

		txCopy.Vin[index].Signature = nil
		txCopy.Vin[index].PubKey = prevTx.Vout[vin.VoutIndex].PubKeyHash

		dataToSign := fmt.Sprintf("%x\n", txCopy)

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign))
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[index].Signature = signature
		txCopy.Vin[index].PubKey = nil
	}
}

//对交易结构体实现String()方法
func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x：", tx.ID))

	for index, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("     Input %d:", index))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.VoutIndex))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for index, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", index))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		address := fmt.Sprintf("%s", PKHashToAddress(output.PubKeyHash))
		lines = append(lines, fmt.Sprintf("       Script: %x(%s)", output.PubKeyHash, address))
	}

	return strings.Join(lines, "\n")
}

/*
	对交易进行修剪，用于签名
	每笔交易输入：Signature：nil   PubKey：nil
 */
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		//fmt.Printf("inputs:%x\n", gobEncode(inputs))
		inputs = append(inputs, TXInput{vin.Txid, vin.VoutIndex, nil, nil})
	}

	for _, vout := range tx.Vout {
		//fmt.Printf("outputs:%x\n", gobEncode(outputs))
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	//txCopyTest := Transaction{tx.ID, nil,nil}
	//fmt.Printf("%x\n", txCopyTest)
	//fmt.Printf("hash %x\n", txCopyTest.Hash())
	return txCopy
}

/*
	实现交易的签名验证
	1、先验证交易输入是否有对应的引用交易
	2、获取交易的修剪版交易
	3、对交易的每笔输入进行签名验证
 */
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin  {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	fmt.Println("Previous transaction is correct")

	//fmt.Printf("%x\n", tx.Serialize())
	fmt.Println(tx)

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for index, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[index].Signature = nil
		txCopy.Vin[index].PubKey = prevTx.Vout[vin.VoutIndex].PubKeyHash

		//fmt.Printf("txCopy.Hash : %x\n", txCopy.Hash())
		//txCopy.ID = txCopy.Hash()
		//txCopy.Vin[index].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen/2)])
		s.SetBytes(vin.Signature[(sigLen/2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen/2)])
		y.SetBytes(vin.PubKey[(keyLen/2):])

		/*fmt.Println(x)
		fmt.Println(y)
		fmt.Println(txCopy.ID)
		fmt.Println(r)
		fmt.Println(s)*/
		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}

		txCopy.Vin[index].PubKey = nil
	}

	return true
}

func (outs TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func DeserializeOutputs(data []byte) TXOutputs  {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		log.Panic(err)
	}

	return outputs
}

func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}