***
# go-CFC
###### *A communication connection agent based on golang development, the goal is simple, efficient, stable, secure, and extensible.*
***
### [简体中文](./README_CN.md)/English
***
## What is go-cfc(Cloud fog connection)?
- This is a simple communication agent, you can think of it as a little toy, it is highly extensible, can surprise (or scare)
- Note that it's not a vpn, and it's much lighter than a vpn, which means it's less cumbersome
- Huh? You're right, it's not very technical. If you can sum it up in one sentence, it's an io.Copy of two net.Conn's, you can write one
***
## What can go-cfc do?
- It can perform Intranet penetration to connect two devices
- The price of a server is not cheap, if the performance is still high, the cost will be heavy, computing to the local device, the server only needs to forward the data delivery, although the response speed is sacrificed, but it will reduce the cost of computing performance
- It has a built-in communication protocol, and you can even use the protocol directly
***
## Why is this thing here?
- I just wanted to ssh my raspberry PI
- I dug up a raspberry PI that I didn't use for a long time, and I wanted to do something with it, but my energy was limited, and I had a job, and I couldn't ssh the Raspberry PI for development whenever I wanted
- Sometimes, I want to ssh my Raspberry PI at work to see things or install software
- But I couldn't ssh my Raspberry PI, because Raspberry PI was at home and I was in the company, so I couldn't directly connect the two Intranet
- I was annoyed, but I didn't want to install someone else's Intranet penetration tool (cumbersome and difficult to understand), so I wrote this to solve the Raspberry PI problem in ssh's home
- But, obviously, it's not just for ssh, and maybe you can use it for other things
- It even helped me save installing unnecessary tools at work, after all, some environment variable Settings are tedious, right?
***
## How to use it?
- It's essentially a tool library, but if you want to use it directly, you can just use the compiled ones [here](./_hook-tcp/asset).
  - ```
    proxy server:
    ./cfc_hook_server -c config.json
    
    device client:
    ./cfc_hook_client -c config.json
    ```
- proxy server(Proxy server)
  - import "github.com/peakedshout/go-CFC/server"
  - ```go
    package example
    
    import "github.com/peakedshout/go-CFC/server"   
    
    func example(){
        server.NewProxyServer("Proxy Server Addr", "32bytes key").Wait()
    } 
    ```
- server listen (Listening connection)
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
- client dial (Request connection)
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
- sub interface (Connection interface)
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
## Features you want to support (later?)
- [x] Support tcp connection
- [x] Support for obtaining one end delay
- [ ] Download files on special links
- [x] Supports subkey encryption
- [ ] Support connection check
- [x] Support for obtaining network speed
- [x] p2p support
***
