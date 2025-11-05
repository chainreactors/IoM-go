# IoM-go

IoM-go 是 Malice Network 的官方 Golang SDK。

## 安装

```bash
go get github.com/chainreactors/IoM-go
```

## 基础用法

### 1. 连接服务器

```go
import (
    "github.com/chainreactors/IoM-go/client"
    "github.com/chainreactors/IoM-go/mtls"
)

// 加载配置
config, _ := mtls.ReadConfig("config.yaml")

// 连接
conn, _ := mtls.Connect(config)
defer conn.Close()

// 初始化服务器状态
server, _ := client.NewServerStatus(conn, config)
```

### 2. 获取会话

```go
// 获取所有活动会话
sessions := server.AlivedSessions()

// 或获取所有会话（包括已断开的）
server.UpdateSessions(true)

// 获取特定会话
session, _ := server.GetOrUpdateSession(sessionId)
```

### 3. 使用会话

```go
// 添加会话以便使用
session := server.AddSession(sessions[0])

// 会话提供 RPC 调用的上下文
ctx := session.Context()

// 检查会话能力
if session.HasDepend("execute") {
    // 会话具有 execute 模块
}
```

### 4. 获取任务

```go
import "github.com/chainreactors/IoM-go/proto/client/clientpb"

tasks, _ := server.Rpc.GetTasks(context.Background(), &clientpb.TaskRequest{
    SessionId: session.SessionId,
})

for _, task := range tasks.Tasks {
    fmt.Printf("任务 %d: %s\n", task.TaskId, task.Type)
}
```

### 5. 管理监听器

```go
// 获取监听器
listeners, _ := server.Rpc.GetListeners(context.Background(), &clientpb.Empty{})

// 列出管道
pipelines, _ := server.Rpc.ListPipelines(context.Background(), &clientpb.Listener{})
```

## 高级用法

### 事件处理

监听服务器的实时事件：

```go
import "github.com/chainreactors/IoM-go/consts"

// 注册事件钩子
server.On(client.EventCondition{
    Type: consts.EventSession,
    Op:   consts.CtrlSessionRegister,
}, func(event *clientpb.Event) (bool, error) {
    fmt.Printf("新会话: %s\n", event.Session.SessionId)
    return true, nil
})

// 启动事件流
eventStream, _ := server.Rpc.Events(context.Background(), &clientpb.Empty{})
go func() {
    for {
        event, _ := eventStream.Recv()
        server.HandlerEvent(event)
    }
}()
```

### 任务回调

异步处理任务结果：

```go
// 注册任务完成回调
server.DoneCallbacks.Store(
    fmt.Sprintf("%s-%d", task.SessionId, task.TaskId),
    func(resp *clientpb.TaskContext) {
        fmt.Printf("任务完成: %s\n", string(resp.Spite.Body))
    },
)
```

### 会话上下文

向会话上下文添加自定义元数据：

```go
// 添加自定义值
sessionWithValue, _ := session.WithValue("key1", "value1", "key2", "value2")

// 获取值
value, _ := sessionWithValue.Value("key1")

// 克隆会话并使用不同的调用者
sdkSession := session.Clone(consts.CalleeSDK)
```

### 观察者模式

监控多个会话：

```go
// 添加观察者
observerId := server.AddObserver(session)

// 获取观察者日志
log := server.ObserverLog(session.SessionId)

// 完成后移除观察者
defer server.RemoveObserver(observerId)
```

### 活动目标管理

```go
// 设置活动会话
server.ActiveTarget.Set(session)

// 获取活动会话
activeSession := server.ActiveTarget.Get()

// 将会话置于后台
server.ActiveTarget.Background()
```

## 示例

查看 [example](./example) 目录获取完整示例：

- **basic_connection** - 连接并获取服务器信息
- **session_management** - 管理会话
- **execute_task** - 获取任务结果
- **event_handling** - 处理实时事件
- **listener_management** - 管理监听器和管道
- **task_callbacks** - 异步任务处理
- **advanced_usage** - 高级模式

```bash
cd example/basic_connection
go run main.go
```

## 链接

- [Malice Network](https://github.com/chainreactors/malice-network)
- [文档](https://chainreactors.github.io/wiki/)
