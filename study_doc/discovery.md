# 看看Discovery中有些什么

```golang
    type Discovery struct {
        c         *conf.Config     // 配置文件
        protected bool             // 受保护模式
        client    http.Client      // 当前节点同步其他节点的客户端
        registry  *registry.Registry
        nodes     atomic.Value
    }

    func New(c *conf.Config) (d *Discovery, cancel context.CancelFunc)  {
        d = &Discovery{
            c:         c,
            protected: c.EnableProtect,
            client:    http.NewClient(c.HttpClient),
            registry:  registry.NewRegistry(c),
        }
        d.nodes.Store(c.Nodes)
        d.syncup()
        go d.nodesproc()
        go d.exitProtect()
        return
    }
}
```

## 配置文件里有什么?

- zones
  - 其他可用区```zone```访问host和其标识
- nodes
  - 同一```discovery```集群的所有node节点地址，包含本```node```
- httpServer的配置
  - addr
  - timeount
- httpClient的配置
  - dial：连接建立的超时时间
  - keepAlive：连接复用保持时间
  - timeout

## 创建一个```discovery```实例的时候做了哪些事

- 将服务启动时的配置文件存储在```discovery.c```
- 构建一个```http.Client```
- 构建一个```register```
- 将配置文件中的```nodes```储存到```discovery```
- ```syncup()```
- ```nodesproc()```
- ```exitProtect()```

## ```atomic.Value```

- 一个线程安全的数据类型，覆盖写入
- CAS指针实现

## ```exitProtect()```

```golang
    func (d *Discovery) exitProtect() {
        // 受保护时间内只允许写不允许读
        time.Sleep(time.Second * 60)
        d.protected = false
    }
```

- ```discovery```创建之后一段时间后```protect```字段置为```false```
- ```discovery```处于```protect```状态时禁止```read```，只允许```write```

## ```syncUp()```

- 使用```http.Client```同步其他节点的数据
- 将```http.Client```读到的数据注册到本节点
- 将本实例置为可用

## ```regSelf()```

## ```nodesproc()```
