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

不过在我们目前的实现中，无法做到完全的去中心化，因为会出现中心化的特点。我们会有三个节点：

   * 一个中心节点。所有其他节点都会连接到这个节点，这个节点会在其他节点之间发送数据，该节点·硬编码·进行代码中。

   * 一个矿工节点。这个节点会在内存池中存储新的交易，当有足够的交易时，它就会打包挖出一个新块。

   * 一个钱包节点。这个节点会被用作在钱包之间发送币。但是与 SPV 节点不同，它存储了区块链的一个完整副本。

场景
---
本文的目标是实现如下场景：

   * 中心节点创建一个区块链。
   * 一个其他（钱包）节点连接到中心节点并下载区块链。
   * 另一个（矿工）节点连接到中心节点并下载区块链。
   * 钱包节点创建一笔交易。
   * 矿工节点接收交易，并将交易保存到内存池中。
   * 当内存池中有足够的交易时，矿工开始挖一个新块。
   * 当挖出一个新块后，将其发送到中心节点。
   * 钱包节点与中心节点进行同步。
   * 钱包节点的用户检查他们的支付是否成功。

这就是比特币中的一般流程。尽管我们不会实现一个真实的 P2P 网络，但是我们会实现一个真实，也是比特币最常见最重要的用户场景。

但缺少了一些像比特币那样的一些关键特性：

0、矿工节点在将交易池的交易进行打包时，没有对多笔交易的输入引用同一个交易进行金额判断。

1、交易ID的生成，目前实现的交易ID值没有包含签名。

2、奖励（reward）。现在挖矿虽然有奖励，但是固定的，而且给发送者，并不是给矿工，同时也没有交易费。


操作方式
-----
首先，在第一个终端窗口中将 NODE_ID 设置为 3000（export NODE_ID=3000）。为了让你知道什么节点执行什么操作，我会使用像 NODE 3000 或 NODE 3001 进行标识。
NODE 3000

创建一个钱包和一个新的区块链：

$ blockchain_go createblockchain -address CENTREAL_NODE

（为了简洁起见，我会使用假地址。）

然后，会生成一个仅包含创世块的区块链。我们需要保存块，并在其他节点使用。创世块承担了一条链标识符的角色（在 Bitcoin Core 中，创世块是硬编码的）

$ cp blockchain_3000.db blockchain_genesis.db 

NODE 3001

接下来，打开一个新的终端窗口，将 node ID 设置为 3001。这会作为一个钱包节点。通过 blockchain_go createwallet 生成一些地址，我们把这些地址叫做 WALLET_1, WALLET_2, WALLET_3.
NODE 3000

向钱包地址发送一些币：

$ blockchain_go send -from CENTREAL_NODE -to WALLET_1 -amount 10 -mine
$ blockchain_go send -from CENTREAL_NODE -to WALLET_2 -amount 10 -mine

-mine 标志指的是块会立刻被同一节点挖出来。我们必须要有这个标志，因为初始状态时，网络中没有矿工节点。

启动节点：

$ blockchain_go startnode

这个节点会持续运行，直到本文定义的场景结束。
NODE 3001

启动上面保存创世块节点的区块链：

$ cp blockchain_genesis.db blockchain_3001.db

运行节点：

$ blockchain_go startnode

它会从中心节点下载所有区块。为了检查一切正常，暂停节点运行并检查余额：

$ blockchain_go getbalance -address WALLET_1
Balance of 'WALLET_1': 10

$ blockchain_go getbalance -address WALLET_2
Balance of 'WALLET_2': 10

你还可以检查 CENTRAL_NODE 地址的余额，因为 node 3001 现在有它自己的区块链：

$ blockchain_go getbalance -address CENTRAL_NODE
Balance of 'CENTRAL_NODE': 10

NODE 3002

打开一个新的终端窗口，将它的 ID 设置为 3002，然后生成一个钱包。这会是一个矿工节点。初始化区块链：

$ cp blockchain_genesis.db blockchain_3002.db

启动节点：

$ blockchain_go startnode -miner MINER_WALLET

NODE 3001

发送一些币：

$ blockchain_go send -from WALLET_1 -to WALLET_3 -amount 1
$ blockchain_go send -from WALLET_2 -to WALLET_4 -amount 1

NODE 3002

迅速切换到矿工节点，你会看到挖出了一个新块！同时，检查中心节点的输出。
NODE 3001

切换到钱包节点并启动：

$ blockchain_go startnode

它会下载最近挖出来的块！

暂停节点并检查余额：

$ blockchain_go getbalance -address WALLET_1
Balance of 'WALLET_1': 9

$ blockchain_go getbalance -address WALLET_2
Balance of 'WALLET_2': 9

$ blockchain_go getbalance -address WALLET_3
Balance of 'WALLET_3': 1

$ blockchain_go getbalance -address WALLET_4
Balance of 'WALLET_4': 1

$ blockchain_go getbalance -address MINER_WALLET
Balance of 'MINER_WALLET': 10
