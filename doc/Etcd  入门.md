# Etcd  入门
## 启动步骤
1. 首先在 github 上根据官方教程下载到本地电脑上来
2. 启动：`nohup ./etcd --listen-client-urls 'http://0.0.0.0:2379' --advertise-client-urls 'http://0.0.0.0:2379' &`
## etcd 核心特性

1. 将数据存储在集群中的高可用的 k-v 存储
2. 允许应用实时监听存储中的K-V变化
3. 能够容忍单点故障，能够应对网络分区

## etcd 原理

### 抽屉原理
抽屉原理也称为大多数原理，一个班级有60人，有一个秘密告诉班上31个人，那么随便挑选31人中肯定有一人知道这个秘密

### etcd 和Raft的关系
- raft 是一个强一致的集群日志同步算法
- etcd 是一个分布式KV存储
- etcd 利用Raft 算法在集群中同步KV存储数据

通俗来说，etcd 负责存储，raft 算法来实现在集群中数据的一致性。将 kv 放入日志中，这些 kv 也就变成强一致的数据了。

### quorum 模型
集群中需要2N+1节点，这样才能产生大多数。

调用过程分为二阶段
1. 当调用者向 leader 发出请求是，leader 不会立即将结果返回给调用者，而是将日志实时复制给 follower。
2. 当复制给N+1节点后，本地进行提交，将结果返回给客户端

![](https://ws3.sinaimg.cn/large/006tKfTcly1g1inedvix0j31mb0u0h6t.jpg)

1. 在完成提交之后，leader 会将提交信息异步通知 给follower，这个 follower 也会完成本地提交
![](https://ws2.sinaimg.cn/large/006tKfTcly1g1ingqbl9rj31rs0u0kf5.jpg)


## 日志格式

![](https://ws2.sinaimg.cn/large/006tKfTcly1g1inhai54nj310t0u07t8.jpg)

从上图可以看出，1到7的日志已经被大多数节点获取(3)，这些数据都可以提交不会丢失。

###Raft 日志概念
- replication:日志在 leader 中生成，向 follower 复制，最终达到各个节点的日志序列最终一致
- term：任期，到达任期后，重新选取产生 leader，其 term 单调递增
- log index:日志行在日志序列中的下标

### Raft 异常场景
![](https://ws1.sinaimg.cn/large/006tKfTcly1g1inpvdz4ej30wy0u0qqf.jpg)

**leader 选举规则**
- 选举 leader 需要半数以上的节点参与
- 节点 commit 日志最多的允许选举为 leader
- commit 日志同样多，则 term，index 越大的允许选举为 leader 

## Raft 工作示例

- 网络分区，当前 leader 无法将数据复制给其他 follower。这个 leader 进行重新启动，在启动的过程中会选举出新的 leader
![](https://ws2.sinaimg.cn/large/006tKfTcly1g1insqpit8j31h00u0qmt.jpg)

- 选举新的 leader，重新进行工作
![](https://ws1.sinaimg.cn/large/006tKfTcly1g1inu895jdj31n80u0h47.jpg)


- 新旧 leader 之间的数据交互，当 old leader 重启完成之后，发现集群中有新的 leader 并且本身的数据落后了，就会获取最新数据替换掉旧的数据。
![](https://ws1.sinaimg.cn/large/006tKfTcly1g1invmebwlj31g10u0ton.jpg)

**Raft 保证**
- 提交成功的请求，一定不会丢
- 各个节点的数据将最终一直

## 交互协议
- 通用HTTP-JSON协议，性能低
- SDK内置GRPC协议，性能高：基于 http2协议之上，使用 protobuffer3序列化库，来做请求和应答的序列化和反序列化。


## 重要特性
- 底层存储是按 key 有序排列，可以顺序遍历
- 因为 key 有序，所有 etcd 天然支持按目录结构高效遍历
- 支持复杂事务，提供类似 if...then...else..的事务能力
- 基于租约机制实现 key 的TTL过期

### key 的有序存储
例如在在寻找某个以 f 开头的目录时，只要 seek 到不大于/feature-flags/的第一个 key，开始向后SCAN即可

![](https://ws1.sinaimg.cn/large/006tKfTcly1g1io2ub51cj31wo0qeqjq.jpg)

### MVCC多版本控制
- 每一个请求都会对应着一个版本号，并且提交的版本在 etcd 中单调递增。
- 同 key 维护多个历史版本，用于实现 watch 机制
- 历史版本过多，可以使用compact 命令删除历史记录

### 监听KV变化
![](https://ws4.sinaimg.cn/large/006tKfTcly1g1io7c1o2kj30ti0j4grh.jpg)
- 通过 watch 机制，可以监听某个Key，或者某个目录(key前缀）的连续变化
- 常用于分布式系统的配置分发，状态同步

#### watch 原理

![](https://ws3.sinaimg.cn/large/006tKfTcly1g1io9sv88xj31jx0u04ig.jpg)

- 当用户请求监听 key=a，rev=1时，watcher 会扫描历史 revision，遍历多版本存储中的数据。
- 当找到对应的数据时，会将这些信息推送给用户

### lease 租约

![](https://ws4.sinaimg.cn/large/006tKfTcly1g1iocd2g3nj31hx0u04fa.jpg)

- SDK通过Grant调用创建一个租约，etcd 会返回一个 id 为5的 lease 给SDK
- SDK使用 put a= 1;lease=5,数据存在 kv 存储引擎中，这个数据 a 和租约 id=5建立关联，等到租约过期之后这个数据 a 就会被删除
- 当SDK还需要这个数据时，需要调用KeepAlive续约