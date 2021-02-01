package BlockInfo

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	"golang.org/x/crypto/ripemd160"
	"log"
)

const version  = byte(0x00)     //定义版本号，一个字节
//const walletFile = "wallet.dat"
const addressChecksumLen  = 4   //定义checksum长度为四个字节

type Wallet struct {
	PrivateKey ecdsa.PrivateKey   	//私钥
	PublicKey []byte				//公钥
}

//通过椭圆曲线加密算法生成私钥，私钥产生公钥
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	publicKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, publicKey
}

//创建钱包下的密钥
func NewWallet() *Wallet  {
	privateKey, publicKey := newKeyPair()
	return &Wallet{privateKey, publicKey}
}

//获取钱包下密钥对应的比特币地址
func (w *Wallet) GetAddress() []byte  {
	//1、将公钥通过ripmd160(sha256(pk))生成20字节公钥哈希
	publicKeyHash := Ripmd160Hash(w.PublicKey)
	//fmt.Printf("PublicKeyHash: %x\n", publicKeyHash)

	//2.在公钥哈希前增加版本号（0x00，代表比特币地址)
	version_publicKeyHash := append([]byte{version}, publicKeyHash...)

	//3.对 版本+公钥哈希 进行双哈希，取前4个字节作为校验码checkSum
	checkSumBytes := CheckSum(version_publicKeyHash)

	//4、对版本号 + PublicKeyHash + checkSum 拼接生成25个字节
	bytes := append(version_publicKeyHash, checkSumBytes...)

	//进行Base58Encode编码，生成易于人们识别的格式
	return Base58Encode(bytes)
}

//通过钱包中的公钥生成公钥哈希  A = RIREMD160(SHA256(Pk))
func Ripmd160Hash(publicKey []byte) []byte  {
	hash256 := sha256.New()
	hash256.Write(publicKey)

	hash := hash256.Sum(nil)

	//hash := sha256.Sum256(publicKey)

	ripemd160 := ripemd160.New()
	ripemd160.Write(hash)

	return ripemd160.Sum(nil)
}

//获取sha256(sha256(前缀+公钥哈希))的前4个字节 作为校验码
func CheckSum(payload []byte) []byte  {
	hash1 := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash1[:])

	return hash2[:addressChecksumLen]
}

//判断比特币地址是否有效
func ValidForAddress(address string) bool  {
	version_publicKeyHash_checkSumBytes := Base58Decode([]byte(address))

	checkSumBytes := version_publicKeyHash_checkSumBytes[len(version_publicKeyHash_checkSumBytes)-addressChecksumLen:]
	//fmt.Println("checkSumBytes: ", checkSumBytes)

	version_publicKeyHash := version_publicKeyHash_checkSumBytes[:len(version_publicKeyHash_checkSumBytes)-addressChecksumLen]

	/* 可提取出公钥哈希
	publicKeyHash := version_publicKeyHash[1:]
	fmt.Printf("PublicKeyHash: %x\n", publicKeyHash)
	*/

	checkBytes := CheckSum(version_publicKeyHash)
	//log.Println(checkSumBytes)
	//log.Println(checkBytes)
	if bytes.Compare(checkSumBytes, checkBytes) == 0 {
		return true
	}

	return false
}