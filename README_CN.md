***
# go-CFC
###### *一个基于golang开发的通信连接代理工具，目标是简单、高效、稳定、安全、拓展的。*
***
### 简体中文/[English](./README.md)
***
## 什么是go-cfc(Cloud fog connection)【云雾连接】？
- 这是一个简单的通信代理玩意，你可以把它当作一个小玩具，它的拓展性很高，能够带来惊喜（或者是惊吓）
- 注意，它并不是vpn，它比vpn更轻量化，这意味着，它并不繁琐
- 嗯？你想的没错，它的技术含量并不高，如果用一句概括它的原理，那就是对两个net.Conn进行了io.Copy，你也可以写一个
***
## go-cfc能做什么？
- 它能够进行内网穿透，将两个设备建立连接
- 一台服务器的价格并不便宜，如果性能还很高，那成本将是沉重的，将计算交付给本地设备，服务器只需要将数据进行转发交付即可，虽然会牺牲响应速度，但这会降低因为计算性能而付出的成本
- 它内置了一套通信协议，你甚至可以直接将该协议进行使用
***
## 为什么会出现这玩意？
- 我只是想ssh我的树莓派
- 我翻出了很久没用的树莓派，我想用它来做一些东西，但我的精力有限，我还有工作，我无法随时ssh树莓派进行开发
- 有时，我想在公司里ssh我的树莓派，用来看一些情况，或者是安装一些软件
- 但我ssh不到我的树莓派，因为树莓派在家里，而我在公司，在两个内网无法直接构成连接
- 我很懊恼，但我不想去装别人的内网穿透工具（繁琐和难以理解），所以我写出了这么一个玩意，用来解决ssh家里的树莓派的问题
- 但，很显然，它的功能不止用于ssh，或许你可以用它来完成一些其他玩意
- 它甚至帮助了我在工作中节省了安装一些不必要的工具，毕竟一些环境变量设置很繁琐，对吧？
***
## 怎么使用？
- 它本质上是一个工具库，但如果想直接使用，可以直接使用[这里](./_hook-tcp/asset)的已经编译好的。
  - ```
    proxy server:
    ./cfc_hook_server -c config.json
    
    device client:
    ./cfc_hook_client -c config.json
    ```
- proxy server(代理服务器)
  - import "github.com/peakedshout/go-CFC/server"
  - ```go
    package example
    
    import "github.com/peakedshout/go-CFC/server"   
    
    func example(){
        server.NewProxyServer("Proxy Server Addr", "32bytes key").Wait()
    } 
    ```
- server listen (监听连接)
  - import "github.com/peakedshout/go-CFC/client"
  - ```go
    package example
    
    import "github.com/peakedshout/go-CFC/client"
    
    func example()  {
        box,_:=client.LinkProxyServer("server name","Proxy Server Addr", "32bytes key")
        defer box.Close()
        box.ListenSubBox(func(sub *client.SubBox){
            defer sub.Close()
        })
        box.Wait()
    }
    ```
- client dial (请求连接)
  - import "github.com/peakedshout/go-CFC/client"
  - ```go
    package example
    
    import "github.com/peakedshout/go-CFC/client"
    
    func example()  {
        box,_:=client.LinkProxyServer("client name","Proxy Server Addr", "32bytes key")
        defer box.Close()
        sub,_:=box.GetSubBox("server name")
        defer sub.Close()
    }
    ```
- sub interface (连接接口)
  - import "github.com/peakedshout/go-CFC/client"
  - ```go
    package example
    
    import "github.com/peakedshout/go-CFC/client"
    
    func example()  {
        //var sub *client.SubBox
        //Read(b []byte) (int, error)
        //Write(b []byte) (int, error)
        //Close() error
        //LocalAddr() net.Addr
        //RemoteAddr() net.Addr
        //SetDeadline(t time.Time) error
        //SetReadDeadline(t time.Time) error
        //SetWriteDeadline(t time.Time) error
        //GetLocalName() string
        //GetRemoteName() string
        //SetDeadlineDuration(timeout time.Duration) bool
        //NewKey(key string) tool.Key
        //GetNetworkSpeedView() tool.NetworkSpeedView
        //GetAllNetworkSpeedView() tool.NetworkSpeedView
        //ReadCMsgCb(fn func(cMsg tool.ConnMsg) (bool, error)) error
        //WriteCMsg(header string, id string, code int, data interface{}) error
        //WriteQueueBytes(b [][]byte) error
    }
    ```
***
## 想支持的功能（后续完成？）
  - [x] 支持tcp连接
  - [x] 支持获取某一端延迟
  - [ ] 支持特殊链路下载文件
  - [x] 支持子key加密
  - [ ] 支持连接校验
  - [x] 支持获取网络速度
  - [x] 支持p2p
***
