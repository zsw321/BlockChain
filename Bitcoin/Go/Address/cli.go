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
func (cli *CLI) createWallet()  {
	wallets, _ := NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}

/*
	创建区块链命令，并将创世纪块的奖励给地址address
	1、判断地址是否合规；
	2、创建一条只包含创世纪块的区块链，生成数据库文件，奖励给地址address
 */
func (cli *CLI) createBlockchain(address string)  {
	if !ValidForAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := CreateBlockchain(address)
	bc.Db.Close()
	fmt.Println("Done!")
}

/*
	显示所有的地址命令
	通过加载wallet.dat文件中保存的私钥-公钥-地址信息，将所有的地址进行遍历
*/
func (cli *CLI) listAddresses()  {
	wallets, err := NewWallets()
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
func (cli *CLI) getBalance(address string)  {
	if !ValidForAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	
	bc := GetBlockchain4db()
	defer bc.Db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-4]
	UTXOs := bc.FindUTXO(pubKeyHash)

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
func (cli *CLI) send(from, to string, amount int)  {
	if !ValidForAddress(from) {
		log.Panic("ERROR: Address is not valid")
	}
	if !ValidForAddress(to) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := GetBlockchain4db()
	defer bc.Db.Close()

	tx := NewTransaction(from, to ,amount, bc)
	bc.MineBlock([]*Transaction{tx})

	fmt.Println("Success!")
}

/*
	打印区块链相关信息命令
	1、通过读取数据库文件从而获取区块链实例（包含指向最后的区块哈希和数据库连接）
	2、遍历区块链区块，输出区块信息
 */
func (cli *CLI) printChain()  {
	bc := GetBlockchain4db()
	defer bc.Db.Close()

	bci := bc.Iterator()
	for {
		block := bci.Next()

		fmt.Printf("============ Block %x ============\n", block.Hash)
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

func (cli *CLI) Run()  {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
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
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses()
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