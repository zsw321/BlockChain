package BlockInfo

/*
	建立一个与程序交互的命令
	addBlock
	printchain
 */

import (
	"BlockChain4Go/tools"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

type CLI struct {}

func (cli *CLI) printUsage()  {
	fmt.Println("Usage:")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  printchain - Print all the blocks of the blockchain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send AMOUNT of coins from FROM address to TO")
}

func (cli *CLI) validateArgs()  {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) createBlockchain(address string)  {
	bc := CreateBlockchain(address)
	bc.Db.Close()
	fmt.Println("Done!")
}

func (cli *CLI) getBalance(address string)  {
	bc := GetBlockchain4db()
	defer bc.Db.Close()

	balance := 0
	UTXOs := bc.FindUTXO(address)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s'：'%d'\n", address, balance)
}

// 发送一笔交易
// 1、生成一笔从from到to的amount数量的转账交易
// 2、根据这笔交易进行区块生成（挖区块）
func (cli *CLI) send(from, to string, amount int)  {
	bc := GetBlockchain4db()
	defer bc.Db.Close()

	tx := NewTransaction(from, to ,amount, bc)
	bc.MineBlock([]*Transaction{tx})

	fmt.Println("Success!")
}

func (cli *CLI) Run()  {
	cli.validateArgs()

	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}

func (cli *CLI) printChain()  {
	bc := GetBlockchain4db()
	defer bc.Db.Close()

	bci := bc.Iterator()
	for {
		block := bci.Next()

		fmt.Printf("Prev. hash：%x\n", block.PrevBlockHash)
		//fmt.Printf("Data：%s\n", block)
		fmt.Printf("Hash：%x\n", block.Hash)

		//timestamp := []byte(strconv.FormatInt(block.Timestamp, 10))
		strTime := tools.UnixTimeToDate(block.Timestamp)
		fmt.Println(string(strTime))

		pow := NewProofOfWork(block)
		fmt.Printf("Pow：%s\n", strconv.FormatBool(pow.Validate()))

		fmt.Println()

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}