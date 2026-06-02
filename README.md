# kit - Go 通用工具库

简洁实用的 Go 工具库，提供常用功能模块，简化项目开发。

## 安装

```bash
go get github.com/tiamxu/kit
```

## 模块概览

| 模块 | 说明 |
|------|------|
| **errors** | 统一错误处理（错误码、sentinel errors） |
| **http** | HTTP 服务器和中间件（基于 Gin） |
| **kafka** | Kafka 消息队列（基于 kafka-go） |
| **log** | 日志记录（基于 zap + lumberjack） |
| **redis** | Redis 客户端封装（含缓存语义：序列化、分布式锁） |
| **sql** | 数据库封装（基于 sqlx，MySQL/PostgreSQL/ClickHouse） |

---

## 快速开始

### 1. 错误处理 (errors)

```go
import "github.com/tiamxu/kit/errors"

// 判断未找到错误
if errors.IsNotFound(err) {
    // 处理未找到
}

// 包装错误（带错误码）
return errors.Wrap("USER_NOT_FOUND", "user id not exists", err)

// 获取错误码
code := errors.GetCode(err)
fmt.Println(code) // "USER_NOT_FOUND"
```

### 2. Redis 客户端 (redis)

```go
import "github.com/tiamxu/kit/redis"

cfg := &redis.Config{
    Address:     "localhost:6379",
    Password:     "",
    DB:           0,
    PoolSize:     20,
    MaxIdle:      15,
    DialTimeout:  5,
    ReadTimeout:  10,
    WriteTimeout: 10,
}

client, err := redis.NewClient(cfg)
if err != nil {
    panic(err)
}

// 基础操作
client.Set(ctx, "key", "value", 10*time.Minute)
val, _ := client.Get(ctx, "key")
client.Delete(ctx, "key")
client.Exists(ctx, "key1", "key2")
client.Expire(ctx, "key", 5*time.Minute)
client.GetTTL(ctx, "key")

// 批量操作
client.MGet(ctx, "key1", "key2")
client.MSet(ctx, map[string]string{"k1": "v1", "k2": "v2"}, 10*time.Minute)

// 原子操作
count, _ := client.Incr(ctx, "counter")
count, _ = client.IncrBy(ctx, "counter", 10)

// 遍历 key（替代 KEYS 命令）
keys, _ := client.Scan(ctx, "prefix:*", 100)

// 对象缓存（JSON 序列化）
client.SetObj(ctx, "user:1", &User{Name: "alice"}, 10*time.Minute)
var user User
found, _ := client.GetObj(ctx, "user:1", &user)

// 分布式锁
lock, err := client.TryLock(ctx, "resource:1", 10*time.Second)
if lock != nil {
    defer lock.Unlock(ctx)
    lock.Refresh(ctx, 5*time.Second) // 续约
}

// 错误处理
if errors.Is(err, redis.ErrNotFound) {
    // key 不存在
}
```

### 3. 数据库 (sql)

基于 sqlx，支持 MySQL、PostgreSQL、ClickHouse。**API 完全兼容 sqlx**，所有 sqlx 方法可直接使用。

```go
import "github.com/tiamxu/kit/sql"

cfg := &sql.Config{
    Driver:   "mysql",
    Database: "test",
    Username: "root",
    Password: "password",
    Host:     "localhost",
    Port:     3306,
    MaxIdleConns:    5,
    MaxOpenConns:    10,
    ConnMaxLifetime: 300,
}

db, err := sql.Connect(cfg)
if err != nil {
    panic(err)
}

// 完全兼容 sqlx - 直接使用 sqlx 的所有方法
var users []User
db.Select(&users, "SELECT * FROM users WHERE age > ?", 18)

var user User
db.Get(&user, "SELECT * FROM users WHERE id = ?", 1)

db.Exec("INSERT INTO users (name, age) VALUES (?, ?)", "alice", 30)

// 事务封装 - 比 sqlx 更简洁
err = db.TransactCallback(func(tx *sqlx.Tx) error {
    tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromID)
    tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", amount, toID)
    return nil
})

// 带 Context 的事务（可取消）
db.TransactCallbackCtx(ctx, func(tx *sqlx.Tx) error {
    tx.ExecContext(ctx, "UPDATE ...")
    return nil
})

// 连接池状态
stats := db.Stats()
fmt.Printf("OpenConnections: %d", stats.OpenConnections)
```

**懒加载初始化（PreDB）：**

启动时不连接数据库，首次使用时才初始化，适合依赖外部服务的场景：

```go
// 创建懒加载实例
predb := sql.NewPreDB()

// 启动时无需连接数据库，应用可以正常启动
// ...

// 首次使用时初始化（如配置加载完成后）
err := predb.Init(&sql.Config{
    Driver:   "mysql",
    Host:     "localhost",
    Database: "test",
    Username: "root",
    Password: "password",
})
if err != nil {
    return err
}

// 获取 DB 实例
db := predb.DB()
if db == nil {
    return errors.New("db not initialized")
}

// 正常使用
var users []User
db.Select(&users, "SELECT * FROM users")
```

**与 sqlx 的区别：**

| 操作 | sqlx | kit sql |
|------|------|---------|
| `db.Select()` | ✅ | ✅ 直接用 |
| `db.Get()` | ✅ | ✅ 直接用 |
| `db.Exec()` | ✅ | ✅ 直接用 |
| `db.Begin()` | ✅ | ✅ 直接用 |
| 事务 | 手动 begin/commit | ✅ `TransactCallback` 自动处理 |

### 4. Kafka 生产者 (kafka)

```go
import "github.com/tiamxu/kit/kafka"

cfg := &kafka.Config{
    Brokers:      []string{"localhost:9092"},
    Topic:        "test-topic",
    MaxRetries:   3,
    RetryInterval: 100 * time.Millisecond,
    BatchTimeout: 100 * time.Millisecond,
    BatchSize:    100,
}

producer, err := kafka.NewKafkaProducer(cfg)
if err != nil {
    panic(err)
}
defer producer.Close()

// 发送消息
err = producer.SendMessage("test-topic", []byte("key"), []byte("value"))
```

### 5. 日志 (log)

基于 zap + lumberjack，支持 stdout 和 file 输出。

```go
import "github.com/tiamxu/kit/log"

// 仅控制台输出
cfg := &log.Config{
    Level:  "info",
    Type:   "stdout",
    Format: "json",
}

// 文件输出（自动滚动）
cfg := &log.Config{
    Level:      "info",
    Type:       "file",
    Format:     "json",
    FilePath:   "logs",
    FileName:   "app.log",
    MaxSize:    100,    // MB
    MaxBackups: 5,
    MaxAge:     30,     // 天
    Compress:   true,
}

// 控制台 + 文件同时输出
cfg := &log.Config{
    Level:      "info",
    Type:       "stdout+file",
    Format:     "json",
    FilePath:   "logs",
    FileName:   "app.log",
}

log.InitLogger(cfg)
defer log.Sync()

// 基础日志
log.Infof("Hello, %s!", "world")
log.Errorf("Error: %v", err)
log.Debugf("debug info: %+v", obj)
log.Warnf("warning: %s", msg)

// 结构化日志
log.WithFields(log.Fields{
    "user_id": 123,
    "action":  "login",
}).Info("user login success")

// 带上下文的日志
log.WithContext(ctx).Infof("request processed")

// 直接使用导出的 Sugar（完全兼容 zap API）
log.Sugar.Infof("hello %s", name)
log.Logger.Info("msg", zap.String("k", "v"))
```

### 6. HTTP 服务器 (http)

基于 Gin，提供中间件和优雅关闭。

```go
import "github.com/tiamxu/kit/http"

cfg := http.ServerConfig{
    Address:         ":8080",
    ReadTimeout:     30 * time.Second,
    WriteTimeout:    30 * time.Second,
    MultipartMemory: 32 << 20, // 32MB
    BodyLimit:       8 << 20,  // 8MB
}

// 创建服务（内部自动启动监听）
srv, err := http.NewServer(cfg)
if err != nil {
    log.Errorf("start server error: %v", err)
    return
}

// 添加路由
srv.GET("/api/users", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"users": []string{"user1", "user2"}})
})

// 优雅关闭
defer srv.Shutdown(context.Background())
```

**自动注册的中间件：**

| 中间件 | 说明 |
|--------|------|
| `RequestIDMiddleware()` | 请求 ID 生成（优先读取 X-Request-ID Header） |
| `AccessLogMiddleware(format)` | 访问日志 |
| `CorsMiddleware(config)` | CORS 跨域配置 |
| `ErrorHandler()` | 统一错误处理 |
| `bodyLimitMiddleware(limit)` | 请求体大小限制 |

**可选添加的中间件：**

| 中间件 | 说明 |
|--------|------|
| `TimeoutMiddleware(timeout)` | 请求超时控制（自动返回 408） |

**日志格式说明：**

默认日志格式（使用 `|` 分隔）：
```
${client_ip} | ${time} | "${method} ${path}" | ${status} | ${bytes_out} | ${user_agent} | ${request_time} | ${request_id} | ${error}
```

| 占位符 | 说明 | 示例 |
|--------|------|------|
| `${client_ip}` | 客户端 IP | `192.168.1.1` |
| `${time}` | 请求时间 | `2026-06-01 15:04:05` |
| `${method}` | HTTP 方法 | `GET` |
| `${path}` | 请求路径 | `/api/users` |
| `${status}` | 响应状态码 | `200` |
| `${bytes_out}` | 响应大小 | `1234` |
| `${user_agent}` | 用户代理 | `Mozilla/5.0` |
| `${request_time}` | 请求耗时 | `0.023s` |
| `${request_id}` | 请求 ID | `abc-123-def` |
| `${error}` | 错误信息（无错误时为空） | |
| `${query}` | 查询参数（可选） | `page=1` |
| `${referer}` | 来源页面（可选） | `https://google.com` |
| `${real_ip}` | 真实 IP（可选，通过 X-Real-IP header） | `10.0.0.1` |
| `${protocol}` | 协议版本（可选） | `HTTP/1.1` |

**日志输出示例：**
```
192.168.1.1 | 2026-06-01 15:04:05 | "GET /api/users" | 200 | 1234 | Mozilla/5.0 | 0.023s | abc-123-def |
```

---

## 配置说明

### Redis 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Address | string | :6379 | Redis 地址 |
| Password | string | "" | 密码 |
| DB | int | 0 | 数据库编号 |
| PoolSize | int | 20 | 连接池大小 |
| MaxIdle | int | 15 | 最大空闲连接 |
| MinIdle | int | 0 | 最小空闲连接 |
| DialTimeout | int | 5 | 连接超时（秒） |
| ReadTimeout | int | 10 | 读取超时（秒） |
| WriteTimeout | int | 10 | 写入超时（秒） |

### 数据库配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Driver | string | - | mysql/postgres/clickhouse |
| Database | string | - | 数据库名 |
| Username | string | - | 用户名 |
| Password | string | - | 密码 |
| Host | string | - | 主机地址 |
| Port | int | 3306/5432/9000 | 端口 |
| MaxIdleConns | int | 5 | 最大空闲连接 |
| MaxOpenConns | int | 10 | 最大打开连接 |
| ConnMaxLifetime | int | 300 | 连接生命周期（秒） |
| ConnMaxIdleTime | int | 60 | 空闲回收时间（秒） |

### Kafka 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Brokers | []string | - | Broker 地址列表 |
| Topic | string | - | 默认主题 |
| MaxRetries | int | 3 | 最大重试次数 |
| RetryInterval | time.Duration | 100ms | 重试间隔 |
| BatchTimeout | time.Duration | 100ms | 批量提交超时 |
| BatchSize | int | 100 | 批量大小 |

### 日志配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Level | string | info | debug/info/warn/error |
| Type | string | stdout | stdout/file/stdout+file |
| Format | string | json | json/console |
| FilePath | string | logs | 日志目录 |
| FileName | string | app.log | 日志文件名 |
| MaxSize | int | 100 | 单文件最大 MB |
| MaxBackups | int | 5 | 保留备份数 |
| MaxAge | int | 30 | 保留天数 |
| Compress | bool | true | 是否压缩 |

---

## 错误码规范

| 前缀 | 模块 |
|------|------|
| `REDIS_*` | redis |
| `SQL_*` | sql |
| `KAFKA_*` | kafka |
| `HTTP_*` | http |

---

## 最佳实践

### 1. 缓存优先查询模式

```go
func GetUser(ctx context.Context, db *sql.DB, client *redis.Client, id int) (*User, error) {
    cacheKey := fmt.Sprintf("user:%d", id)

    // 1. 先查缓存
    var user User
    found, err := client.GetObj(ctx, cacheKey, &user)
    if err != nil {
        return nil, err
    }
    if found {
        return &user, nil
    }

    // 2. 缓存未命中，查数据库
    err = db.Get(&user, "SELECT * FROM users WHERE id = ?", id)
    if err != nil {
        if sql.IsNoRows(err) {
            return nil, nil
        }
        return nil, err
    }

    // 3. 写入缓存
    client.SetObj(ctx, cacheKey, &user, 10*time.Minute)
    return &user, nil
}
```

### 2. 数据库事务 + 缓存一致性

```go
func UpdateUser(ctx context.Context, db *sql.DB, client *redis.Client, id int, name string) error {
    err := db.TransactCallback(func(tx *sqlx.Tx) error {
        _, err := tx.Exec("UPDATE users SET name = ? WHERE id = ?", name, id)
        return err
    })
    if err != nil {
        return err
    }

    // 删除缓存，确保一致性
    cacheKey := fmt.Sprintf("user:%d", id)
    client.Delete(ctx, cacheKey)
    return nil
}
```

### 3. 带超时的数据库操作

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

err = db.TransactCallbackCtx(ctx, func(tx *sqlx.Tx) error {
    tx.ExecContext(ctx, "UPDATE orders SET status = ? WHERE user_id = ?", "paid", userID)
    return nil
})
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        // 处理超时
    }
}
```

---

## 依赖版本

- Go: 1.24+
- Redis: go-redis/v9
- Database: sqlx
- HTTP: Gin
- Log: zap + lumberjack
- Kafka: kafka-go