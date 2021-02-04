实现的功能
----
在UTXO目录功能的基础上实现了区块链的简单网络通信功能。   
1、底层实现了区块链的区块数据进行数据库（bolt）的保存;   
2、实现钱包创建功能，包括私钥-公钥-地址；   
3、实现了Coinbase交易和普通交易的生成，普通交易的每笔输入都要通过私钥进行签名，并保存进输入的签名字段，同时增加奖励；   
4、实现UTXO集，即未花费输出保存进行数据库功能，用于构建交易、查询余额时查询；   
5、实现了Merkle Root构造生成根哈希；   
6、将生成的交易打包区块，并通过工作量证明机制实现区块的生成（难度固定），并保存进数据库中；  
7、增加了节点之间的区块同步功能。

不过在我们目前的实现中，无法做到完全的去中心化，因为会出现中心化的特点。我们会有三个节点，每个节点对应本地一个端口：

   * 一个中心节点。所有其他节点都会连接到这个节点，这个节点会在其他节点之间发送数据，该节点端口`硬编码`进代码中。  

   * 一个矿工节点。这个节点会在内存池中存储新的交易，当有足够的交易时，它就会打包挖出一个新块。

   * 一个钱包节点。这个节点会被用作在钱包之间发送币。但是与 SPV 节点不同，它存储了区块链的一个完整副本。

场景
---
本文的目标是实现如下场景：

   * 中心节点创建一个区块链。    
   1、创建钱包地址和包含创世纪块的区块链，并把数据库文件备份一下；  
   2、给钱包地址发送比特币；  
   3、启动节点，对本地端口3000进行监听，等待其他节点连接;  
   6、接收到钱包节点（localhost:3001）的连接，对消息进行解析，处理version消息，将当前节点保存的区块链区块高度与消息中的高度进行比较，当前节点的区块高度更大，则给钱包地址(localhost:3001)发送version消息（包含当前区块高度、当前节点地址localhost:3000）,同时将钱包地址添加进knowNodes（[]strings{localhost:3000,localhost:3001}）,等待其他节点连接；  
   8、接收到钱包节点（localhost:3001）的连接，对消息进行解析，处理getblocks消息，获取本地数据库中所有的区块哈希，则给钱包地址发送inv消息（包含当前节点地址localhost:3000、kind为block、区块哈希），等待其他节点连接；  
   10、接收到钱包节点（localhost:3001）的连接，对消息进行解析，处理getdata消息，根据消息中的区块哈希查询本地数据库，获取区块，则给钱包地址发送block消息（包含当前节点地址localhost:3000、区块序列化数据），等待其他节点连接；  
   12、重复步骤10，直到区块发送完毕，等待其他节点连接；  
   14-2、矿工节点同步区块，同时将矿工节点地址添加进knowNodes（[]strings{localhost:3000,localhost:3001,localhost:3002}）    
   16、接收到钱包节点（localhost:3001）的连接，对消息进行解析，处理tx消息，从消息中取出序列化后交易并反序列化，保存进交易池中（map[string]Transactionyins 映射结构），对knowNodes切片进行遍历，如切片的值不是本地地址（localhost:3000）和消息来源地址（localhost:3001）,则向其（只剩下localhost:3002）发送inv消息（包含当前节点地址localhost:3000、kind为tx、交易ID），等待其他节点连接；  
   18、接收到挖矿节点（localhost:3002）的连接，对消息进行解析，处理getdata消息，取出消息中的交易ID，从交易池中取出交易ID对应的交易信息，给挖矿节点（localhost:3002）发送tx消息（包含当前节点地址localhost:3000、交易序列化）,等待其他节点连接；  
   21、接收到钱包节点（localhost:3001）的连接，对消息进行解析，处理tx消息，从消息中取出序列化后交易并反序列化，保存进交易池中（map[string]Transactionyins 映射结构），重复不止步骤16-18，等待其他节点连接；  
   23、接收到挖矿节点（localhost:3002）的连接，对消息进行解析，处理inv消息，从消息中取出区块哈希，向挖矿节点发送getdata消息（包含当前节点地址localhost:3000、kind为block、区块哈希），等待其他节点连接；    
   25、接收挖矿节点（localhost:3002）的连接，对消息进行解析，处理block消息，从消息中取出区块序列化数据，并反序列化，添加到本地数据库中，并更新本地UTXO集索引，等待其他节点连接；  
   
   
   * 一个其他（钱包）节点连接到中心节点并下载区块链。  
   2、创建钱包地址；  
   4、将步骤1保存的包含创世纪块的数据库文件复制为本地端口3001对应的数据库；  
   5、启动本地节点（端口3001），则会给中心节点发送version消息（包含当前区块高度、当前节点地址localhost:3001），然后处于监听状态，等待连接；  
   7、接收到中心节点（localhost:3000）的连接，对消息进行解析，处理version消息，将当前节点保存的区块链区块高度与消息中的高度进行比较，消息中的区块高度较大（即中心节点下的区块高度大），则给中心地址（localhost:3000）发送getblocks消息（当前节点地址localhost:3001），等待其他节点连接;  
   9、接收到中心节点（localhost:3000）的连接，对消息进行解析，处理inv消息，先保存消息中的区块哈希切片，取切片中第一个索引下的区块哈希blockhash，则给中心地址发送getdata消息（包含当前节点地址localhost:3001、kind为block、blockhash）,并将消息中剩下的区块哈希进行保存（blocksInTransit），等待其他节点连接；  
   11、接收到中心节点（localhost:3000）的连接，对消息进行解析，处理block消息，将消息中的区块序列化数据进行反序列化，将区块增加到本地数据库中，判断步骤9保存的blocksInTransit切片长度是否大于0（是否还有区块哈希），若有，则获取切片中第一个索引号下区块哈希blockhash，则给中心地址发送getdata消息（包含当前节点地址localhost:3001、kind为block、blockhash）,并将消息中剩下的区块哈希进行保存（blocksInTransit），等待其他节点连接；  
   13、重复步骤11，直到区块接收完毕，等待其他节点；  
 
   * 另一个（矿工）节点连接到中心节点并下载区块链。  
   14-1、与中心节点下载区块的步骤跟钱包节点一致；
   * 钱包节点创建一笔交易。  
   15、构造一笔交易（send命令不带参数-mine，代表不用由钱包节点立即生成区块），给中心节点发送tx消息（包含当前节点地址localhost:3001、交易序列化数据）；  
   20、继续构造一条交易（send命令不带参数-mine，代表不用由钱包节点立即生成区块），给中心节点发送tx消息（包含当前节点地址localhost:3001、交易序列化数据）；  
   
   * 矿工节点接收交易，并将交易保存到内存池中。  
   17、接收到中心节点（localhost:3000）的连接，对消息进行解析，处理inv消息，取出消息中的交易ID，向中心节点地址（localhost:3000）发送getdata消息（包含当前节点地址localhost:3002、kind为tx、交易ID），等待其他节点连接；  
   19、接收到中心节点（localhost:3000）的连接，对消息进行解析，处理tx消息，取出消息中的交易序列化数据，并反序列化的交易保存进交易池中，判断交易池中的交易数是否大于等于2同时挖矿地址是否存在，当前交易池的交易只有1条，故等待其他节点连接；  
   
   * 当内存池中有足够的交易时，矿工开始挖一个新块。  
   * 当挖出一个新块后，将其发送到中心节点。  
    22、继续步骤17-19，判断交易池中的交易数是否大于等于2同时挖矿地址是否存在，当前交易池的交易已有2条，对其进行验证，同时构建奖励交易，将它们用于打包区块，然后重新生成UTXO索引，删除交易池中的交易，向中心节点发送inv消息（包含当前节点地址localhost:3002、kind为block，新增区块的哈希），等待其他节点连接；  
   24、接收到中心节点（localhost:3000）的连接，对消息进行解析，处理getdata消息，取出消息中的区块哈希，从数据库中找到区块，向中心节点发送block消息（当前节点地址localhost:3002、区块序列化数据），等待其他节点连接；
   * 钱包节点与中心节点进行同步。  
   26、钱包节点启动节点，从中心节点同步区块。
   * 钱包节点的用户检查他们的支付是否成功。
  
综述所示：钱包节点与中心节点的通信如下：
![](https://raw.githubusercontent.com/zsw321/BlockChain/master/Bitcoin/Go/4-Network/update.png)


这就是比特币中的一般流程。尽管我们不会实现一个真实的 P2P 网络，但是我们会实现一个真实，也是比特币最常见最重要的用户场景。

但缺少了一些像比特币那样的一些关键特性：

0、矿工节点在将交易池的交易进行打包时，没有对多笔交易的输入引用同一个交易进行金额判断。

1、交易ID的生成，目前实现的交易ID值没有包含签名。

2、奖励（reward）。现在挖矿虽然有奖励，但是固定的，而且给发送者，并不是给矿工，同时也没有交易费。


操作方式
-----
首先，在第一个终端窗口中将 NODE_ID 设置为 3000（set NODE_ID=3000）。为了让你知道什么节点执行什么操作，我会使用像 NODE 3000 或 NODE 3001 进行标识。
NODE 3000

创建一个钱包和一个新的区块链：

$ test.exe createblockchain -address %CENTREAL_NODE%

（为了简洁起见，我会使用假地址。）

然后，会生成一个仅包含创世块的区块链。我们需要保存块，并在其他节点使用。创世块承担了一条链标识符的角色（在 Bitcoin Core 中，创世块是硬编码的）

$ cp blockchain_3000.db blockchain_genesis.db 

NODE 3001

接下来，打开一个新的终端窗口，将 node ID 设置为 3001。这会作为一个钱包节点。通过 test.exe createwallet 生成一些地址，我们把这些地址叫做 WALLET_1, WALLET_2, WALLET_3、WALLET_4.

NODE 3000

向钱包地址发送一些币：

$ test.exe send -from %CENTREAL_NODE% -to %WALLET_1% -amount 10 -mine
$ test.exe send -from %CENTREAL_NODE% -to %WALLET_2% -amount 10 -mine

-mine 标志指的是块会立刻被同一节点挖出来。我们必须要有这个标志，因为初始状态时，网络中没有矿工节点。

启动节点：

$ test.exe startnode

这个节点会持续运行，直到本文定义的场景结束。
NODE 3001

启动上面保存创世块节点的区块链：

$ cp blockchain_genesis.db blockchain_3001.db

运行节点：

$ test.exe startnode

它会从中心节点下载所有区块。为了检查一切正常，暂停节点运行并检查余额：

$ test.exe getbalance -address %WALLET_1%
Balance of 'WALLET_1': 10

$ test.exe getbalance -address %WALLET_2%
Balance of 'WALLET_2': 10

你还可以检查 CENTRAL_NODE 地址的余额，因为 node 3001 现在有它自己的区块链：

$ test.exe getbalance -address %CENTRAL_NODE%
Balance of 'CENTRAL_NODE': 10

NODE 3002

打开一个新的终端窗口，将它的 ID 设置为 3002，然后生成一个钱包。这会是一个矿工节点。初始化区块链：

$ cp blockchain_genesis.db blockchain_3002.db

启动节点：

$ test.exe startnode -miner %MINER_WALLET%

NODE 3001

发送一些币：

$ test.exe send -from %WALLET_1% -to %WALLET_3% -amount 1
$ test.exe send -from %WALLET_2% -to %WALLET_4% -amount 1

NODE 3002

迅速切换到矿工节点，你会看到挖出了一个新块！同时，检查中心节点的输出。
NODE 3001

切换到钱包节点并启动：

$ test.exe startnode

它会下载最近挖出来的块！

暂停节点并检查余额：

$ test.exe getbalance -address %WALLET_1%
Balance of 'WALLET_1': 9

$ test.exe getbalance -address %WALLET_2%
Balance of 'WALLET_2': 9

$ test.exe getbalance -address %WALLET_3%
Balance of 'WALLET_3': 1

$ test.exe getbalance -address %WALLET_4%
Balance of 'WALLET_4': 1

$ test.exe getbalance -address %MINER_WALLET%
Balance of 'MINER_WALLET': 10
