# 服务间的运行

## 整个服务的组件:
````
type Discovery struct {
	c         *conf.Config          // 启动配置文件
	protected bool                  // 保护模式
	client    *http.Client          // 本实例的http客户端
	registry  *registry.Registry    // 实例的增删改查以及维护
	nodes     atomic.Value          // 理解为与其他节点通信的客户端
}
````

## 启动服务
- 根据配置文件初始化nodes、http客户端、注册中心
- 同步信息
  - 从除本节点之外的所有节点fecth实例并且注册到本节点
  - 将本节点的状态置为启动
- 维护当前服务
  - 在当前服务时上注册自己
  - 启动一个协程
    - 给本服务续租
    - 监听服务程序关闭信号（用于cancel（下线）本服务）
- 维护discovery
  - 使用poll监听所有register中的Discovery服务
    - 当监测到变化时，重载nodes

## Nodes

- 数据结构:
``` 
type Nodes struct {
	nodes    []*Node            // 同机房（zone）的其他实例
	zones    map[string][]*Node // 其他机房（zone）的其他实例
	selfAddr string             
}

type Node struct {
	c *conf.Config              // 配置文件

	// 与制定实例通信的客户端与其对应的url
	client       *http.Client
	pRegisterURL string         
	registerURL  string
	cancelURL    string
	renewURL     string
	setURL       string

	addr      string
	status    model.NodeStatus
	zone      string            // 机房名
	otherZone bool              // 是否是其他机房
}
```
- 每一个Node可以理解为与其对应的节点的通信客户端
- Nodes管理所有本Discovery服务与其他Discovery服务通信的客户端

## 服务之间如何进行同步
以服务启动为例介绍服务之间信息同步的流程

同机房流程简介：<br>
现在假设Discovery叫D,三个Discovery实例依次启动，分别是D1、D2、D3，<br>
它们的register分别是R1、R2、R3，<br>
D1、D2、D3分别运行在M1、M2、M3三台机器。<br>
指向D1、D2、D3的客户端分别是C1、C2、C3
- D1启动
  - D1拥有的客户端[C1,C2,C3]
  - D2、D3都未启动，所以此时R1拥有的实例:[D1]
  - D1拥有的客户端被重载，重载后:[C1]
- D2启动
  - D1拥有的客户端[C1,C2,C3]
  - D2向D1获取实例信息注册到R2,R2拥有的实例:[D2,D1]
  - D2拥有的客户端被重载，重载后:[C2,C1]
  - D2向R2注册自己，注册完成后通过C1同步到R1
  - D1监听到R1上新增实例后重载客户端:[C1,C2]
  - R1拥有的实例:[D1,D2]，D1拥有的客户端[C1,C2]
  - R2拥有的实例:[D2,D1]，D2拥有的客户端[C2,C1]
- D3启动
  - D3拥有的客户端[C1,C2,C3]
  - D3向D1、D2获取实例信息注册到R3,R3拥有的实例:[D1,D2]
  - D3向R3注册自己
  - R3拥有的实例:[D1,D2,D3],D3客户度端重载为:[C1,C2,C3]
  - D3通过C1、C2同步信息到R1、R2
  - R1拥有的实例:[D1,D2,D3],R2拥有的实例:[D2,D1,D3]
  - D1、D2分别监控到R1、R2的实例变化
  - D1拥有的客户端:[C1,C2,C3],D2拥有的客户端:[C2,C1,C3]
- 最终
  - R1拥有的实例:[D1,D2,D3],D1拥有的客户端[C1,C2,C3]
  - R2拥有的实例:[D2,D1,D3],D2拥有的客户端[C2,C1,C3]
  - R3拥有的实例:[D2,D3,D1],D3拥有的客户端[C2,C3,C1]
### 多机房
- Nodes.zones存储的是其他机房的节点信息
- 每一个zone中有n个实例
- 每一次节点内部的信息改变依靠的就是zones向其他机房同步信息
- 但是同步并不会每一次向每一个客户端发送同步，而是确保每一个机房有一个实例被同步到变化
- 收到变化的节点再通过自己同步到机房的其他节点

### 负载均衡算法
该项目使用的是客户端负载均衡，算法如下：<br>
- 将拉到的实例按照它们所属的机房分组
- 计算出每一组的权重总和(zoneTotalWeight[zone])
- 每一组权重总和累乘（comMulti）
- 每一个机房本的权重(zoneWeight[zone])
- （zoneWeight * comMulti / zoneTotalWeight）得到每一个机房的校准值（fixWeight[zone]）
- 每一个实例的最终权重是自身的权重乘以机房校准值

## 信息同步中的一些细节
- 只有“源请求”才会在被处理之后通过节点的客户端转发。举个例子：
  - 有D1、D2、D3三个实例
  - D1收到一个实例I1的注册请求
  - D1完成注册后转发到D2、D3，D2，D3在完成注册后不会再次转发
- 机房与机房之间也类似。举个例子：
  - 房间Z1有有D1、D2、D3三个实例，，Z1挂载的ZA实例是DA，房间ZA有DA、DB、DC三个实例，ZA挂载的Z1节点是D1
  - D1收到一个实例I1的注册请求
  - D1完成注册后转发到D2、D3，同时D1会把数据同步到DA
  - 对于房间ZA，来自D1的同步请求不是“源请求”，对于房间DA来自D1的同步是“源请求”，所以DA会把这条注册信息同步给DB、DC，但是房间ZA不会再通过D1向Z1回传

## 总结服务之间的信息同步
- 单点发起，全网广播，增量修改，最终一致
