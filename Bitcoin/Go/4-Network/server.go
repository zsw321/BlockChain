package BlockInfo

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

var nodeListenAddress string
var miningAddress string
var knownNodes = []string{"localhost:3000"}
var blocksInTransit = [][]byte{}
var mempool = make(map[string]Transaction)

type addr struct {
	AddrList []string
}

type block struct {
	AddrFrom string
	Block []byte
}

type getblocks struct {
	AddrFrom string
}

type getdata struct {
	AddrFrom string
	Type string
	ID []byte
}

type inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type tx struct {
	AddFrom     string
	Transaction []byte
}

type verzion struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func bytesToCommand(bytes []byte) string  {
	var command []byte
	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}
	return fmt.Sprintf("%s", command)
}

func extractCommand(requeset []byte) []byte  {
	return requeset[:commandLength]
}

func StartServer(nodeID, minerAddress string)  {
	nodeListenAddress = fmt.Sprintf("localhost:%s", nodeID)
	fmt.Println("myListenAddress:"+nodeListenAddress)
	miningAddress = minerAddress

	ln, err := net.Listen(protocol, nodeListenAddress)
	if err != nil {
		log.Panic(err)
	}

	defer ln.Close()

	bc := GetBlockchain4db(nodeID)
	if nodeListenAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for  {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleConnection(conn, bc)
	}
}

func handleConnection(conn net.Conn, bc *Blockchain)  {
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}

	command := bytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

func handleAddr(request []byte)  {
	var buff bytes.Buffer
	var payload addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	requestBlocks()
}

func requestBlocks()  {
	for _, node := range knownNodes {
		sendGetBlocks(node)
	}
}

func handleInv(request []byte, bc *Blockchain)  {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Received inventory with %d %s\n", len(payload.Items), payload.Type)
	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if mempool[hex.EncodeToString(txID)].ID == nil {
			sendGetData(payload.AddrFrom, "tx", txID)
		}
	}

}

func handleTx(request []byte, bc *Blockchain)  {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx

	//fmt.Printf("tx hash %x", tx.Hash())
	//fmt.Println(tx)
	if !bc.VerifyTransaction(&tx) {
		fmt.Println("transactions are invalid! Waiting for new ones...")
		return
	}

	if nodeListenAddress == knownNodes[0] {
		for _, node := range knownNodes {
			fmt.Println("node: "+node)
			fmt.Println("nodeListenAddress: "+nodeListenAddress)
			if node != nodeListenAddress && node != payload.AddFrom {
				sendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(mempool) >= 2 && len(miningAddress) > 0 {
			MineTransaction:
				var txs []*Transaction
				for id := range mempool {
					tx := mempool[id]
					if bc.VerifyTransaction(&tx) {
						txs = append(txs, &tx)
					}
				}
				if len(txs) == 0 {
					fmt.Println("All transactions are invalid! Waiting for new ones...")
					return
				}

				cbTx := NewCoinbaseTX(miningAddress, "")
				txs = append(txs, cbTx)

				newBlock := bc.MineBlock(txs)
				UTXOSet := UTXOSet{bc}
				UTXOSet.Reindex()

				fmt.Println("New block is mined!")

				for _, tx := range txs {
					txID := hex.EncodeToString(tx.ID)
					delete(mempool, txID)
				}

				for _, node := range knownNodes {
					if node != nodeListenAddress {
						sendInv(node, "block", [][]byte{newBlock.Hash})
					}
				}

				if len(mempool) > 0 {
					goto MineTransaction
				}
		}
	}
}

func handleGetBlocks(request []byte, bc *Blockchain)  {
	var buff bytes.Buffer
	var payload getblocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := bc.GetBlockHashes()
	sendInv(payload.AddrFrom, "block", blocks)
}

func handleBlock(request []byte, bc *Blockchain)  {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := DeserializeBlock(blockData)

	fmt.Println("Recevied a new block!")
	bc.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)
	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := UTXOSet{bc}
		UTXOSet.Reindex()
	}
}

func handleVersion(request []byte, bc *Blockchain)  {
	var buff bytes.Buffer
	var payload verzion

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, bc)
	}

	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}
}

func sendGetData(address, kind string, id []byte)  {
	payload := gobEncode(getdata{nodeListenAddress, kind, id})
	request := append(commandToBytes("getdata"), payload...)
	fmt.Println("command getdata")
	sendData(address, request)
}

func sendGetBlocks(address string)  {
	payload := gobEncode(getblocks{nodeListenAddress})
	request := append(commandToBytes("getblocks"), payload...)
	fmt.Println("command getblocks")
	sendData(address, request)
}

func sendInv(address, kind string, items [][]byte)  {
	inventory := inv{nodeListenAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)
	fmt.Println("command inv")
	sendData(address, request)
}

func sendVersion(addr string, bc *Blockchain)  {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(verzion{nodeVersion, bestHeight, nodeListenAddress})

	requst := append(commandToBytes("version"), payload...)
	fmt.Println("command version")
	sendData(addr, requst)
}

func sendBlock(addr string, b *Block)  {
	data := block{nodeListenAddress, b.Serialize()}
	payload := gobEncode(data)

	request := append(commandToBytes("block"), payload...)
	fmt.Println("command block")
	sendData(addr, request)
}

func sendData(addr string, data []byte)  {
	fmt.Println("Begin connect address: "+addr)
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		//var updateNodes []string

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func sendTx(addr string, tnx *Transaction)  {
	data := tx{nodeListenAddress, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	sendData(addr, request)
}

func handleGetData(request []byte, bc *Blockchain)  {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := bc.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		sendBlock(payload.AddrFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := mempool[txID]

		sendTx(payload.AddrFrom, &tx)
	}

}

func gobEncode(data interface{}) []byte  {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func nodeIsKnown(addr string) bool  {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}
	return false
}


