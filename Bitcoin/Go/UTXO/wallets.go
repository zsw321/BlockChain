package BlockInfo

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type Wallets struct {
	Wallets map[string]*Wallet
}

/*
	创建钱包集对象
	读取钱包文件wallet.dat（没有则创建）的内容进行初始化
	内容包含地址与钱包的映射数据结构map[string]*Wallet
 */
func NewWallets() (*Wallets, error) {
	//1、声明钱包集结构体，并将包含的映射数据进行make声明
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	//2、通过加载钱包文件wallet.dat（没有则创建），并初始化钱包集结构体
	err := wallets.LoadWalletsFromFile()

	return &wallets, err
}

//读取钱包文件wallet.dat（没有则创建）的内容，并初始化进ws.Wallets
func (ws *Wallets) LoadWalletsFromFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets
	return nil
}

/*
	在当前钱包集基础上创建新的私钥-公钥-地址信息
	1、创建一组钱包信息，即私钥-公钥-地址
	2、将创建的钱包信息添加到钱包集ws.Wallets中
 */
func (ws *Wallets) CreateWallet()  string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = wallet

	return address
}

/*
	获取钱包集中所有的地址
 */
func (ws *Wallets) GetAddresses() []string {
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	return addresses
}

//获取当前钱包集下地址为address下的钱包信息，即私钥-公钥对
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

/*
	将当前钱包集的内容序列化，并保存进行钱包文件wallet.dat
 */
func (ws Wallets) SaveToFile()  {
	var content bytes.Buffer

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}