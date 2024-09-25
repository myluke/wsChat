# WebSocket 聊天应用

这是一个使用Go语言后端和JavaScript前端实现的简单WebSocket聊天应用。

## 功能特点

- 实时聊天：使用WebSocket进行实时双向通信
- 多会话支持：用户可以创建和切换不同的聊天会话
- 时间戳：每条消息都显示发送时间
- 服务器回声：服务器会对每条消息进行回复

## 技术栈

- 后端：Go + gorilla/websocket
- 前端：HTML + JavaScript + WebSocket API

## 如何运行

### 前提条件

- 安装Go（版本1.16+）
- 安装gorilla/websocket包：`go get github.com/gorilla/websocket`

### 运行服务器

1. 克隆此仓库
2. 进入项目目录
3. 运行命令：`go run main.go`
4. 服务器将在8880端口启动

### 运行客户端

1. 在浏览器中打开`index.html`文件
2. 确保WebSocket连接地址正确（默认为`ws://localhost:8880/ws`）

## 使用说明

1. 打开客户端页面后，点击"新建会话"按钮创建一个新的聊天会话
2. 在输入框中输入消息，点击"发送"按钮或按回车键发送消息
3. 您可以创建多个会话，并通过点击左侧的会话列表来切换不同的会话

## 项目结构

- `main.go`：后端服务器代码
- `index.html`：前端页面和JavaScript代码
- `go.mod`：Go模块定义文件

## 注意事项

- 这是一个简单的演示项目，没有实现用户认证和消息持久化
- 在实际生产环境中，请确保添加适当的安全措施和错误处理

## 贡献

欢迎提交问题和改进建议！如果您想为这个项目做出贡献，请提交pull request。

## 许可

此项目采用MIT许可证。详情请见LICENSE文件。