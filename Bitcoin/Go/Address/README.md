
在Transaction目录实现的功能基础上，实现了区块链的钱包地址生成功能，包括创建钱包、生成私钥-公钥-地址。 
0、增加钱包地址（公钥-私钥）功能；
1、底层实现了区块链的区块数据进行数据库（bolt）的保存; 
2、实现了Coinbase交易和普通交易的生成，普通交易的每笔输入都要通过私钥进行签名，并保存进输入的签名字段；
3、将生成的交易打包区块，并通过工作量证明机制实现区块的生成（难度固定），并保存进数据库中； 

但缺少了一些像比特币那样的一些关键特性：

1、交易ID的生成，目前实现的交易ID值没有包含签名。

2、奖励（reward）。现在挖矿是肯定无法盈利的！，同时也没有交易费。

3、UTXO 集。获取余额需要扫描整个区块链，而当区块非常多的时候，这么做就会花费很长时间。并且，如果我们想要验证后续交易，也需要花费很长时间。而 UTXO 集就是为了解决这些问题，加快交易相关的操作。

4、内存池（mempool）。在交易被打包到块之前，这些交易被存储在内存池里面。在我们目前的实现中，一个块仅仅包含一笔交易，这是相当低效的。

$ test.exe createwallet
Your new address: 1AmVdDvvQ977oVCpUqz7zAPUEiXKrX5avR

$ test.exe createwallet
Your new address: 1NE86r4Esjf53EL7fR86CsfTZpNN42Sfab

$ test.exe createblockchain -address 1AmVdDvvQ977oVCpUqz7zAPUEiXKrX5avR
000000122348da06c19e5c513710340f4c307d884385da948a205655c6a9d008

Done!

$ test.exe send -from 1AmVdDvvQ977oVCpUqz7zAPUEiXKrX5avR -to 1NE86r4Esjf53EL7fR86CsfTZpNN42Sfab -amount 6
0000000f3dbb0ab6d56c4e4b9f7479afe8d5a5dad4d2a8823345a1a16cf3347b

Success!

$ test.exe getbalance -address 1AmVdDvvQ977oVCpUqz7zAPUEiXKrX5avR
Balance of '1AmVdDvvQ977oVCpUqz7zAPUEiXKrX5avR': 4

$ test.exe getbalance -address 1NE86r4Esjf53EL7fR86CsfTZpNN42Sfab
Balance of '1NE86r4Esjf53EL7fR86CsfTZpNN42Sfab': 6

$test.exe printchain
============ Block 000000829c337959d230c1f63589091c369c0d9263a4dc575a5a5eab4bb3d425 ============
Prev. block: 000001d57b04680b20a75e4b3012dbb607375057c69db1ad477dbe11aecb8b30
PoW: true

--- Transaction 5b1dd02e06124bf6e177cdae986c4c8773d19a525cb95d4d60745f230c30fb14：
     Input 0:
       TXID:      df58e9ca9f4ad5b06a9d98b2b07aecfca1097a7570b5419e480cf75d40f5a55a
       Out:       0
       Signature: 5c4fa2cf348e7f95ae3177f63c006b2814d281b21091357939281b8c2339e7e2afc22806528cc998735199bac9fd7c5e3e7151ff3dcbb23159dbe2cca5a2b20c
       PubKey:    4f71a3a5cddf6a02e5052a1e246756e2a8b001176079a1894b82e0848ce71356313d315053e3cd154287302e844ebb7801df2d073cdbced4ca8837e4b6a9d418
     Output 0:
       Value:  5
       Script: 610ff69333c510d6f9105a1f82cab9c0ddc48245
     Output 1:
       Value:  5
       Script: f0038f2b603cdb1785b9a175aa6d4b9e192f2520


============ Block 000001d57b04680b20a75e4b3012dbb607375057c69db1ad477dbe11aecb8b30 ============
Prev. block:
PoW: true

--- Transaction df58e9ca9f4ad5b06a9d98b2b07aecfca1097a7570b5419e480cf75d40f5a55a：
     Input 0:
       TXID:
       Out:       -1
       Signature:
       PubKey:    5468652054696d65732030332f4a616e2f32303039204368616e63656c6c6f72206f6e206272696e6b206f66207365636f6e64206261696c6f757420666f722062616e6b73
     Output 0:
       Value:  10
       Script: f0038f2b603cdb1785b9a175aa6d4b9e192f2520

