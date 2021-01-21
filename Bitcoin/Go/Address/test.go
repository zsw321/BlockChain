package main

import "BlockChain4Go/BlockInfo"

func main()  {
	/*
	bc := BlockInfo.NewBlockChain()

	bc.AddBlock("Send 1 BTC to Ivan")
	bc.AddBlock("Send 2 more BTC to Ivan")

	for _, block := range bc.Blocks {
		fmt.Printf("Prev. hash：%x\n", block.PrevBlockHash)
		fmt.Printf("Data：%s\n", block.Data)
		fmt.Printf("Hash：%x\n", block.Hash)

		//timestamp := []byte(strconv.FormatInt(block.Timestamp, 10))
		strTime := tools.UnixTimeToDate(block.Timestamp)
		fmt.Println(string(strTime))

		pow := BlockInfo.NewProofOfWork(block)
		fmt.Printf("Pow：%s\n", strconv.FormatBool(pow.Validate()))

		fmt.Println()
	}*/

	/*bc := BlockInfo.NewBlockChain()
	defer bc.Db.Close()*/

	cli := BlockInfo.CLI{}

	cli.Run()
}