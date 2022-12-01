- What is CFC(Cloud fog connection)?
  - CFC is a small tool that can broker data connections and surprise (or scare) people.
- What can CFC do?
  - It does everything that net.conn can do, it simply brokers the data of the two conn's, enabling them to establish a virtual connection beyond the original conn service
- What are the advantages of CFC? Or how does it work?
  - It has Intranet penetration (emphasis)
  - Or, you can use it to perform proxy services for both the server and the client
  - You can use it to ssh your raspberry PI
- Why is this thing here?
  - I just wanted to ssh my raspberry PI
  - I dug up a raspberry PI that I didn't use for a long time, and I wanted to do something with it, but my energy was limited, and I had a job, and I couldn't ssh the Raspberry PI for development whenever I wanted
  - Sometimes, I want to ssh my raspberry PI in the company to see what's going on, or to get my hands dirty and develop Raspberry PI
  - But I couldn't ssh my Raspberry PI, because Raspberry PI was at home and I was in the company, so I couldn't directly connect the two Intranet
  - I was annoyed, but I didn't want to install someone else's Intranet penetration tool (cumbersome and difficult to understand), so I wrote this to solve the Raspberry PI problem in ssh's home
  - But, obviously, it's not just for ssh, and maybe you can use it for other things
  - I just want to ssh my raspberry PI, although the effect of ssh is not very satisfactory (it seems to be the terminal problem of Raspberry PI, I don't know, it is a headache).
- How to use it?
  - ct(Transfer server, proxy server)
    - import `import "github.com/peakedshout/go-CFC/server"`
    - ```
       server.NewServer("IP", ":Port", "32byte")
      ```
  - server(Listening connection)
    - import `import "github.com/peakedshout/go-CFC/client"`
    - ```
      c, _ := client.LinkLongConn("sshServer", "IP", ":Port", "32byte")
      defer c.Close()
      c.ListenSubConn(func(sub *client.SubConnContext) {
          defer sub.Close()
          conn:=sub.GetConn()
      })
      ```
  - client(Request a connection)
    - import `import "github.com/peakedshout/go-CFC/client"`
    - ```
      c, _ := client.LinkLongConn("sshClient", "IP", ":Port", "32byte")
      defer c.Close()
      sub, _ := c.GetSubConn("sshServer")
      defer sub.Close()
      conn:=sub.GetConn()
      ```
  - More use visible [_example](./_example)
- Features you want to support (later?)
  - [√] Support for tcp connections
  - [×] Support udp connection
  - [√] Support for obtaining one end delay
  - [×] Download files on special links 
  - [×] Supports subkey encryption
  - [×] Support connection verification
