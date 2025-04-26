## 关于

这是一个为了 [Nekops](https://nekops.app) 定制的嵌入式 SSH 客户端，带有一些传统 SSH 客户端不支持的功能。

请注意，为了实现交互执行了一些特殊处理，这个客户端并不适合作为直接交互使用（即使或许可以）。如果您有使用 SSH 的需求，请使用 OpenSSH 。

## 事件 （暂定）

每个事件由 `\x02` 字符起始， `\x03` 字符结束。

如果事件拥有载荷，那么会以 `\x1f` 字符作为分割，前半部分为事件名（纯文本字符串），后半部分为事件载荷（JSON 格式的字符串）。

当前的事件列表如下：

|   事件   |  事件名  | 是否拥有载荷 | 载荷格式                                                | 含义                                                         |
| :------: | :------: | :----------: |-----------------------------------------------------| ------------------------------------------------------------ |
| SSH 开始 | sshStart |      否      | -                                                   | 预启动阶段结束，上下文(stdin/stdout/stderr)完全交给 SSH 会话 |
| 主机密钥 | hostKey  |      是      | { h: string, s?: string[], o?: string, fp: string } | 首次连接到某主机，或主机的密钥发生变化                       |

具体的事件信息您也可以参阅 `events.go` 文件中的描述。

## 服务端公钥验证

由于 Windows 平台上的 known_hosts 文件使用 CRLF (\r\n) 换行，而 *nix 平台下的换行符为 LF (\n)，为确保跨平台兼容性，这个客户端统一使用 LF 作为换行符。

为避免对您现有的 known_hosts 文件造成损害，这个客户端不再会读写您默认目录（ ~/.ssh/ ）下的 known_hosts 文件。

如果您需要启用服务器公钥验证功能，请使用 `-o UserKnownHostsFile=...` 选项指定 known_hosts 文件。如果您未指定该参数，则默认不会验证服务器公钥。

## 信息

与一般 SSH 不同的是，这个客户端加入了这些新的功能：

1. 捕获 `\e[8;{rows};{cols}t` 格式的 ANSI 转义序列，用于提示远端服务器关于窗口大小的变更事件（经由 stdin 输入）

## 致谢

- 基础流程参考 [A Simple Cross-Platform SSH Client in 100 Lines of Go](https://medium.com/better-programming/a-simple-cross-platform-ssh-client-in-100-lines-of-go-280644d8beea)
- 跳板机逻辑 [Mr_Pink's answer - Go x/crypto/ssh -- How to establish ssh connection to private instance over a bastion node](https://stackoverflow.com/questions/35906991/go-x-crypto-ssh-how-to-establish-ssh-connection-to-private-instance-over-a-ba/35924799#35924799)
