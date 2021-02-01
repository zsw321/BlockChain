package BlockInfo

/*
	建立一个与程序交互的命令
	addBlock
	printchain
 */

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

type CLI struct {}

func (cli *CLI) printUsage()  {
	fmt.Println("Usage:")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  createwallet - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  listaddresses - Lists all addresses from the wallet file")
	fmt.Println("  printchain - Print all the blocks of the blockchain")
	fmt.Println("  reindexutxo - Rebuilds the UTXO set")
	fmt.Println("  printutxo - print the UTXO set")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send AMOUNT of coins from FROM address to TO")
}

func (cli *CLI) validateArgs()  {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

/*
	创建钱包命令
	这里钱包用于保存地址—私钥及公钥对的map映射
	每调用一次创建钱包命令，就会生成一组  私钥-公钥-地址  map[string]*Wallet
	并将钱包的数据保存进行wallet.dat文件，有了该文件，就有了地址里面的私钥-公钥对，即可进行交易签名
 */
func (cli *CLI) createWallet(nodeID string)  {
	wallets, _ := NewWallets(nodeID)
	address := wallets.CreateWallet()
	wallets.SaveToFile(nodeID)

	fmt.Printf("Your new address: %s\n", address)
}

/*
	创建区块链命令，并将创世纪块的奖励给地址address
	1、判断地址是否合规；
	2、创建一条只包含创世纪块的区块链，生成数据库文件，奖励给地址address
 */
func (cli *CLI) createBlockchain(address, nodeID string)  {
	if !ValidForAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := CreateBlockchain(address, nodeID)
	defer bc.Db.Close()

	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	fmt.Println("Done!")
}

/*
	显示所有的地址命令
	通过加载wallet.dat文件中保存的私钥-公钥-地址信息，将所有的地址进行遍历
*/
func (cli *CLI) listAddresses(nodeID string)  {
	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}

}

/*
	获取指定地址的余额命令
	1、先判断地址是否合规
	2、通过读取数据库文件从而获取区块链实例（包含指向最后的区块哈希和数据库连接）
	3、将地址转换成对应的公钥哈希
	4、通过公钥哈希查找所有的UTXO交易输出集
	5、遍历并叠加UTXO的币数
 */
func (cli *CLI) getBalance(address, nodeID  string)  {
	log.Println("Address: "+address)
	if !ValidForAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	
	bc := GetBlockchain4db(nodeID)
	UTXOSet := UTXOSet{bc}
	defer bc.Db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s'：'%d'\n", address, balance)
}

/*
	发送一笔转账命令
	1、判断发送地址、接收地址是否合规；
	2、通过读取数据库文件从而获取区块链实例（包含指向最后的区块哈希和数据库连接）
	3、构建一条交易，实现从from到to的转账
	4、将构建的交易打包进区块（目前没有奖励）
 */
func (cli *CLI) send(from, to, nodeID string, amount int, mineNow bool)  {
	log.Println("From Address: "+from)
	if !ValidForAddress(from) {
		log.Panic("ERROR: From's Address is not valid")
	}

	log.Println("To Address: "+to)
	if !ValidForAddress(to) {
		log.Panic("ERROR: To's Address is not valid")
	}

	bc := GetBlockchain4db(nodeID)
	UTXOSet := UTXOSet{bc}

	defer bc.Db.Close()

	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	tx := NewUTXOTransaction(&wallet, to ,amount, &UTXOSet)
	if mineNow {
		cbTx := NewCoinbaseTX(from, "")
		txs := []*Transaction{cbTx,tx}

		newBlock := bc.MineBlock(txs)
		UTXOSet.Update(newBlock)
	} else {
		if !bc.VerifyTransaction(tx) {
			fmt.Println("transactions are invalid! Waiting for new ones...")
			return
		}

		sendTx(knownNodes[0], tx)
	}


	/*cbTx1 := NewCoinbaseTX(to, "")
	cbTx2 := NewCoinbaseTX(to, "")
	cbTx3 := NewCoinbaseTX(to, "")*/


	fmt.Println("Success!")
}

/*
	打印区块链相关信息命令
	1、通过读取数据库文件从而获取区块链实例（包含指向最后的区块哈希和数据库连接）
	2、遍历区块链区块，输出区块信息
 */
func (cli *CLI) printChain(nodeID string)  {
	bc := GetBlockchain4db(nodeID)
	defer bc.Db.Close()

	bci := bc.Iterator()
	for {
		block := bci.Next()

		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Height: %d\n", block.Height)
		fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)
		pow := NewProofOfWork(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")

		if len(block.PrevBlockHash) == 0 {
			break
		}

	}
}

/*
	重新建立UTXO集
	1、获取区块链实例
	2、重新生成UTXO集
 */
func (cli *CLI) reindexUTXO(nodeID string)  {
	bc := GetBlockchain4db(nodeID)
	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CLI) printUTXOSet(nodeID string)  {
	bc := GetBlockchain4db(nodeID)
	UTXOSet := UTXOSet{bc}
	UTXOSet.PrintUTXO()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CLI) startNode(nodeID, minerAddress string)  {
	fmt.Printf("Starting node %s\n", nodeID)
	if len(minerAddress) > 0 {
		if ValidForAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}

	StartServer(nodeID, minerAddress)
}

func (cli *CLI) Run()  {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!")
		os.Exit(1)
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	printUTXOCmd := flag.NewFlagSet("printutxoset", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")

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
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printutxoset":
		err := printUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress, nodeID)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}
	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}
	if printUTXOCmd.Parsed() {
		cli.printUTXOSet(nodeID)
	}
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}

		cli.send(*sendFrom, *sendTo, nodeID, *sendAmount, *sendMine)
	}
	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			os.Exit(1)
		}
		cli.startNode(nodeID, *startNodeMiner)
	}
}