- 什么是CFC(Cloud fog connection)【云雾连接】？
  - CFC是一个能够代理数据连接的一个小工具，它能给人带来惊喜（或者是惊吓）
- CFC能干什么？
  - net.conn能干的事情，它都能干，它只是代理两个conn的数据，使它们能够建立一个虚拟的连接，不再仅仅局限于原本的conn服务
- CFC的优势是什么？或者说它能怎么用？
  - 它能内网穿透（重点）
  - 或者说，你可以用它来完成服务端和客户端的代理服务
  - 你可以用来ssh你的树莓派
- 为什么出现这玩意？
  - 我只是想ssh我的树莓派
  - 我翻出了很久没用的树莓派，我想用它来做一些东西，但我的精力有限，我还有工作，我无法随时ssh树莓派进行开发
  - 有时，我想在公司里ssh我的树莓派，用来看一些情况，或者是摸鱼去开发树莓派
  - 但我ssh不到我的树莓派，因为树莓派在家里，而我在公司，在两个内网无法直接构成连接
  - 我很懊恼，但我不想去装别人的内网穿透工具（繁琐和难以理解），所以我写出了这么一个玩意，用来解决ssh家里的树莓派的问题
  - 但，很显然，它的功能不止用于ssh，或许你可以用它来完成一些其他玩意
  - 我只是想ssh我的树莓派，尽管ssh出来的效果不是很满意（貌似是树莓派的终端问题，我不清楚，很头疼）
- 怎么使用？
  - ct（中转服务器、代理服务器）
    - import `import "github.com/peakedshout/go-CFC/server"`
    - ```
       server.NewServer("IP", ":Port", "32位字母")
      ```
  - server（监听连接）
    - import `import "github.com/peakedshout/go-CFC/client"`
    - ```
      c, _ := client.LinkLongConn("sshServer", "IP", ":Port", "32byte")
      defer c.Close()
      c.ListenSubConn(func(sub *client.SubConnContext) {
          defer sub.Close()
          conn:=sub.GetConn()
      })
      ```
  - client（请求连接）
    - import `import "github.com/peakedshout/go-CFC/client"`
    - ```
      c, _ := client.LinkLongConn("sshClient", "IP", ":Port", "32byte")
      defer c.Close()
      sub, _ := c.GetSubConn("sshServer")
      defer sub.Close()
      conn:=sub.GetConn()
      ```
  - 更多使用可见 [_example](./_example)
- 想支持的功能（后续完成？）
  - [√] 支持tcp连接 
  - [×] 支持udp连接
  - [√] 支持获取某一端延迟 
  - [×] 支持特殊链路下载文件 
  - [×] 支持子key加密 
  - [×] 支持连接校验
