package BlockInfo

import (
	"bytes"
	"fmt"
	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// 字符组转Base58，对数据进行Base58Check编码
func Base58Encode(input []byte) []byte {

	//fmt.Printf("Base58Encode编码前输入的十六进制：%x\n", input)
	//fmt.Printf("输入的十进制：%d\n", input)
	//fmt.Println("Base58Encode编码前的格式：")

	var result []byte

	x := big.NewInt(0).SetBytes(input)
	//fmt.Printf("Base58Encode编码前输入的十进制：%d\n", x)

	base := big.NewInt(int64(len(b58Alphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}

	//fmt.Printf("字节数组反转前的Address: %s\n", result)

	//若不反转，则最高为在最低位，最低位在最高位
	ReverseBytes(result)

	for  _, b := range input {
		if b == 0x00 {
			//规定前缀为0x00、其Base58结果的前缀为1
			result = append([]byte{b58Alphabet[0]}, result...)
		}else {
			break
		}
	}

	return result
}

func PKHashToAddress(publicKeyHash []byte) []byte {
	version_publicKeyHash := append([]byte{version}, publicKeyHash...)

	//3.对 版本+公钥哈希 进行双哈希，取前4个字节作为校验码checkSum
	checkSumBytes := CheckSum(version_publicKeyHash)

	//4、对版本号 + PublicKeyHash + checkSum 拼接生成25个字节
	bytes := append(version_publicKeyHash, checkSumBytes...)

	//进行Base58Encode编码，生成易于人们识别的格式
	return Base58Encode(bytes)
}

// Base58转字节数组，解码
func Base58Decode(input []byte) []byte  {
	result := big.NewInt(0)
	zeroBytes := 0
	for _, b := range input{
		if b != b58Alphabet[0]{
			break
		}
		zeroBytes++
	}
	payload := input[zeroBytes:]
	for _, b := range payload {
		//fmt.Println("payload: ", b)
		charIndex := bytes.IndexByte(b58Alphabet, b)
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(charIndex)))
	}
	//fmt.Printf("解码后的十六进制：%d\n", result)

	//将大整数转换为[]byte切片类型
	decodeed := result.Bytes()
	decodeed = append(bytes.Repeat([]byte{byte(0x00)}, zeroBytes), decodeed...)
	return decodeed
}

//字节数组反转
func ReverseBytes(data []byte)  {
	for i, j := 0, len(data)-1; i<j; i,j=i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

/*
	PrivateKey WIF 转换成 对应的16进制
	WIF（非压缩的）：私钥经过Base58Check编码  前缀0x80(128-)、和32位的CheckSum(4字节)
	1、将WIF转换成大数，再转成字节切片；
	2、去掉第一个字节的前缀、最后4个字节的CheckSum，得到就是私钥的内容

 */

//版本1
func PrivateKeyOfWIFToHex(data []byte) []byte {
	//1.过来第一个字符5
	//payload := data[1:]

	//转换成大数
	bigResult := big.NewInt(0)
	for _, b := range data {
		charIndex := bytes.IndexByte(b58Alphabet, b)
		bigResult.Mul(bigResult, big.NewInt(58))
		bigResult.Add(bigResult, big.NewInt(int64(charIndex)))
	}

	//将大整数转换为[]byte切片类型
	decodeed := bigResult.Bytes()
	pKHex := decodeed[1: len(decodeed)-4]

	fmt.Printf("十六进制: %x\n", pKHex)
	fmt.Printf("十进制: %d\n", pKHex)

	return pKHex
}

//版本2
func getPrivateKeyFromWIF(wifPrivteKey string) []byte  {
	rawData := []byte(wifPrivteKey)
	base58DecodedData := Base58Decode(rawData)
	fmt.Printf("十六进制: %x\n", base58DecodedData)
	return base58DecodedData[1:len(base58DecodedData)-4]
}



