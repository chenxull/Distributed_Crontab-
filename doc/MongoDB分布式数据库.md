# MongoDB分布式数据库
- 文档数据库，基于二进制JSON存储文档
- 高性能，高可用，直接加机器即可解决扩展性问题
- 支持丰富的CRUD操作：聚合统计，坐标检索

## 启动

下载安装[地址](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-ubuntu/)

1. 完成安装之后，首先选择位置建立数据库位置`mkdir mydata`,数据文件会存储在这里
2. 启动`nohup mongod --dbpath=./mydata & `
3. 启动之后使用`mongo`来连接，这个命令默认连接到本机的 MongoDB 数据库。

## 原理
- Mongod 单机版数据库
- Replica Set :复制集，由多个M ongod 组成的高可用存储单元
- Sharding：分布式集群，由多个Replica Set 组成的可扩展集群

### 架构
![](https://ws4.sinaimg.cn/large/006tKfTcly1g1jhi6dfw0j319e0u0dx4.jpg)

- 默认采用wiredTiger高性能存储引擎
- 基于journaling log 宕机恢复,数据会先写入到日志中，日志存储在内存中，等到一定时间在存储到磁盘中去。如果数据丢失，就从 log 文件中去查询

####Replica Set 架构

![](https://ws1.sinaimg.cn/large/006tKfTcly1g1jhl9w91uj313j0u016n.jpg)
- 至少3个节点组成，其中一个充当 arbiter
- 主从基于oplog复制同步  类比于 mysql binlog

#### Sharding 架构

![](https://ws2.sinaimg.cn/large/006tKfTcly1g1jhn818chj311d0u0k9l.jpg)
- mongos 作为代理，路由请求到特定的 shard
- 3个 mongd 节点组成 config server，保存数据元信息
- 每一个 shard 是一个 replica set ，可无限扩容 

### Collection 分片
![](https://ws3.sinaimg.cn/large/006tKfTcly1g1jhqfashoj31be0toqhh.jpg)

- collection 自动分裂多个 chunk
- 每个 chunk 被自动负载均衡到不同的 shard
- 每个 shard 可以保证其上的chunk 高可用，应为shard 是一个 replica Set 集群

**Collection 的切分规则是什么？**
按照 range 拆分 chunk ，不过按照这种分配的方式，同一时间写入的数据容易被分配到同一个 chunk 上，数据分配不均衡。
![](https://ws1.sinaimg.cn/large/006tKfTcly1g1jhtt8t5bj31ly0rutpr.jpg)

- Shard key 可以是单索引字段或者联合索引字段
- 超过16MB的 chunk 会被一分为二
- 一个 collectionde的所有 chunk 首尾相连，构成整个表

可以使用按 hash 拆分 chunk，就可以避免上述情况
![](https://ws4.sinaimg.cn/large/006tKfTcly1g1jhy6sjk8j31oi0u0198.jpg)
- shard key 必须是哈希索引字段
- shard key 经过hash 函数打散，避免写入热点
- 支持预分配 chunk，避免运行时chunk 的分裂，影响写入

**按照非 shard key 进行查询，请求被扇出给所有 shard。因为分片是按照 x 去做的，x 的数据会被分散到各个 shard 中，按照 x 去请求数据，返回的是分散的 shard**

## 文档数据库 schema free
无需提前创建各种表单，直接往数据库里放什么就有什么

![](https://ws1.sinaimg.cn/large/006tKfTcly1g1jgc5fghoj31n60j2486.jpg)

### 与 mysql 对比
![](https://ws2.sinaimg.cn/large/006tKfTcly1g1jgcq7t4jj31dg0u0h0t.jpg)

## 用法
### 选择数据库
- 例举数据库： `show  databases`
- 选择数据库： `use my_db`

数据库无需创建，直接是一个命名空间

### 创建collection
- 例举数据表：`show databases`
- 简历数据表： `db.createCollection("my_collection")`

数据表schema free,无需定义字段

### 插入 document
用法： `db.my_collection.insertOne({uid:1000,name:"chenxu",likes:["football","game"]})`

- 插入时可以嵌套层级BSON（二进制的JSON）
- 文档的ID会自动生成，无需指定

### 查询 document
`db.my_collect.find(likes:'football',name: {$in:['xiaoming','libai']}}).sort({uid:1})`

- 可以基于任意BSON层级过滤
- 支持的功能和 mysql 相当

### 更新 document
![](https://ws1.sinaimg.cn/large/006tKfTcly1g1jgvim1pxj31qq0mk15t.jpg)

### 删除 document
![](https://ws2.sinaimg.cn/large/006tKfTcly1g1jgszp6i1j31q20hateq.jpg)

### 创建 index
`db.my_collection.creatIndex({uid:1,name:-1})`

可以指定建立索引时的正反顺序


## 聚合

![](https://ws1.sinaimg.cn/large/006tKfTcly1g1jgxl19zhj31dd0u0to1.jpg)

