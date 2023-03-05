# 从main函数开始

## main函数代码

```golang

    func main() {
        flag.Parse()
        if err := conf.Init(); err != nil {
            log.Error("conf.Init() error(%v)", err)
            panic(err)
        }
        log.Init(conf.Conf.Log)
        dis, cancel := discovery.New(conf.Conf)
        http.Init(conf.Conf, dis)
        // init signal
        c := make(chan os.Signal, 1)
        signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
        for {
            s := <-c
            log.Info("discovery get a signal %s", s.String())
            switch s {
            case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
                cancel()
                time.Sleep(time.Second)
                log.Info("discovery quit !!!")
                return
            case syscall.SIGHUP:
            default:
                return
            }
        }
    }

```

- 执行README中的完整的命令目前在我的机器上执行失败，去掉后面的选项```-alsologtostderr```就能成功启动，这个选项干啥的暂时先不管

```shell
./discovery -conf discovery.toml -alsologtostderr
```

- ```main```函数做的事情有:(1)读取配置文件。(2)启动discovert。(3)监听系统信号。

## 信号处理

- ```SIGQUIT, SIGTERM, SIGINT```和```SIGHUP```分别是什么，有什么异同？
  - SIGHUP:
    - 内核驱动发现终端（或伪终端）关闭，给终端对应的控制进程（bash）发 SIGHUP
    - bash 收到 SIGHUP 后，会给各个作业（包括前后台）发送 SIGHUP，然后自己退出
    - 前后台的各个作业收到来自 bash 的 SIGHUP 后退出（如果程序会处理 SIGHUP，就不会退出）

  - SIGQUIT:键入Ctrl+\键位(退出键)
  - SIGTERM:kill命令
  - SIGINT:Ctrl+C
  - SIGKILL:kill -9

- 监听管道的长度为何不是0
  - ```Notify(c chan<- os.Signal, sig ...os.Signal)```第一个参数chan的长度至少要有1，否则会出现丢失监听信号的问题。
    - 例子:

```golang
    // 这段代码。执行之后，在5s内无路按了多少次Ctrl+C五秒后都不会立即退出程序
    func main() {
        c := make(chan os.Signal)
        signal.Notify(c, os.Interrupt)
        time.Sleep(5 * time.Second)
        // Block until a signal is received.
        s := <-c
        fmt.Println("Got signal:", s)
    }
 ```
