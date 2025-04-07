
## 事件 （暂定）

每个事件由 `\x02` 字符起始， `\x03` 字符结束。

如果事件拥有载荷，那么会以 `\x1f` 字符作为分割，前半部分为事件名（纯文本字符串），后半部分为事件载荷（JSON 格式的字符串）。

当前的事件列表如下：

|    事件    | 状态  |     事件名     | 是否拥有载荷 | 载荷格式                                 | 含义                                          |
|:--------:| :---: | :------------: | :----------: |--------------------------------------|---------------------------------------------|
|  SSH 开始  |   ✅️   |    sshStart    |      否      | -                                    | 预启动阶段结束，上下文(stdin/stdout/stderr)完全交给 SSH 会话 |
|   主机密钥   |   🚧   |   hostKey   |      是      | { h: string, s: string[], o: string, k: string } | 首次连接到某主机，或主机的密钥发生变化                         |

具体的事件信息您也可以参阅 `events.go` 文件中的描述。

## 信息

与一般 SSH 不同的是，这个客户端加入了这些新的功能：

1. 捕获 `\e[8;{rows};{cols}t` 格式的 ANSI 转义序列，用于提示远端服务器关于窗口大小的变更事件（经由 stdin 输入）

## 致谢

- 基础流程参考 [A Simple Cross-Platform SSH Client in 100 Lines of Go](https://medium.com/better-programming/a-simple-cross-platform-ssh-client-in-100-lines-of-go-280644d8beea)
- 交互机制参考 [Yevgeniy Brikman's answer - How can I send terminal escape sequences through SSH with Go?](https://stackoverflow.com/questions/28921409/how-can-i-send-terminal-escape-sequences-through-ssh-with-go/37088088#37088088)
- 跳板机逻辑 [Mr_Pink's answer - Go x/crypto/ssh -- How to establish ssh connection to private instance over a bastion node](https://stackoverflow.com/questions/35906991/go-x-crypto-ssh-how-to-establish-ssh-connection-to-private-instance-over-a-ba/35924799#35924799)
