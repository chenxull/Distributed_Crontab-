[![996.ICU](https://img.shields.io/badge/link-996.icu-red.svg)](https://996.icu)
# 开发记录


## 2019年3月30日

### 实现功能
1. 实现提交任务的Etcd 的 API接口
2. 实现删除任务的 API接口
3. 实现展示 etcd 中任务目录的API接口
4. 实现杀死任务的API接口


## 项目结构
  

### master
控制节点，当用户提交任务后，master 节点首先对这些请求进行路由处理，然后将任务分配给不同的 worker 节点进行执行
#### 架构

#### 功能
- 给 web 后台提供 http api，用于管理 job
### worker

### common


## BUG

1. 删除 job 功能存在问题，经过测试无法获取传入的 job name。