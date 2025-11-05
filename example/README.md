# IoM-go 示例

本目录包含 IoM-go SDK 的完整使用示例。

## 运行示例

每个示例都在独立的目录中：

```bash
cd <示例目录>
go run main.go
```

或从 example 目录运行：

```bash
go run ./<示例目录>/main.go
```

## 示例列表

### 基础示例

**basic_connection** - 连接服务器并获取基本信息
```bash
cd basic_connection && go run main.go
```
演示如何建立连接、初始化服务器状态、获取客户端、监听器和会话列表。

**session_management** - 会话管理
```bash
cd session_management && go run main.go
```
演示如何获取会话列表、查询会话详情、检查会话状态和模块。

**execute_task** - 获取任务结果
```bash
cd execute_task && go run main.go
```
演示如何获取会话的任务列表和任务详情。

**listener_management** - 监听器管理
```bash
cd listener_management && go run main.go
```
演示如何管理监听器和管道，包括列出、注册、启动和停止。

### 高级示例

**event_handling** - 事件处理
```bash
cd event_handling && go run main.go
```
演示如何监听服务器的实时事件流。

**task_callbacks** - 任务回调
```bash
cd task_callbacks && go run main.go
```
演示如何使用回调系统异步处理任务结果。

**advanced_usage** - 高级用法
```bash
cd advanced_usage && go run main.go
```
演示高级特性：会话上下文、克隆、观察者模式、活动目标管理等。

## 配置

所有示例使用相对路径 `../../../server/admin_127.0.0.1.auth` 加载配置文件。

如需使用其他配置文件，修改示例中的路径：

```go
config, err := mtls.ReadConfig("path/to/your/config.yaml")
```

## 常见模式

### 基础连接模式

```go
config, _ := mtls.ReadConfig("config.yaml")
conn, _ := mtls.Connect(config)
defer conn.Close()
server, _ := client.NewServerStatus(conn, config)
```

### 会话操作模式

```go
sessions := server.AlivedSessions()
session := server.AddSession(sessions[0])
ctx := session.Context()
```

### RPC 调用模式

```go
tasks, _ := server.Rpc.GetTasks(ctx, &clientpb.TaskRequest{
    SessionId: session.SessionId,
})
```

## 依赖管理

本目录使用统一的 `go.mod` 管理所有示例的依赖：

```bash
go mod tidy
```
