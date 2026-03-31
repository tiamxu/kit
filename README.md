# kit

通用工具类库，提供多种常用功能模块，简化Go项目开发。

## 功能模块

| 模块 | 说明 |
|------|------|
| **cache** | 缓存语义层（序列化、压缩、批量操作、分布式锁） |
| **es** | Elasticsearch 客户端封装 |
| **http** | HTTP 服务器和中间件（基于 Gin） |
| **kafka** | Kafka 消息队列封装（基于 kafka-go） |
| **llm** | 大语言模型客户端封装（支持阿里云） |
| **log** | 日志记录（基于 zap + lumberjack，支持 stdout、file、kafka 多输出） |
| **ops** | 运维相关功能 |
| **redis** | 纯 Redis 客户端封装（连接池、基础命令） |
| **sql** | 数据库操作封装（基于 sqlx，支持 MySQL、PostgreSQL、ClickHouse） |
| **vectorstore** | 向量存储封装（支持 Milvus、Qdrant） |

## 架构设计

```
┌─────────────────────────────────────────────┐
│              业务代码                         │
├─────────────────────────────────────────────┤
│         cache 包（缓存语义层）                 │
│  · 序列化/反序列化 + Gzip 压缩                │
│  · 批量操作 MGet/MSet                        │
│  · 对象缓存 GetObj/SetObj                    │
│  · 分布式锁 TryLock/Unlock                   │
│  · 原子操作 Incr/IncrBy                      │
├─────────────────────────────────────────────┤
│         redis 包（Redis 客户端层）             │
│  · 连接池管理、超时配置                        │
│  · 纯 Redis 命令封装                          │
│  · MGet/MSet/Scan 等批量操作                  │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│         log 包（日志层）                       │
│  · 基于 zap + lumberjack                      │
│  · 多输出：stdout / file / kafka              │
│  · 组合输出：stdout+file / stdout+kafka       │
│  · Kafka 异步写入，buffer 满不阻塞业务         │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│         sql 包（数据库层）                     │
│  · 基于 sqlx，支持 MySQL/PostgreSQL/ClickHouse│
│  · 连接池管理、配置校验                        │
│  · 事务 TransactCallback                     │
└─────────────────────────────────────────────┘
```

## 快速开始

### 安装

```bash
go get github.com/tiamxu/kit
```

### 使用示例

#### 1. Redis 客户端

纯 Redis 客户端，提供连接池管理和基础命令封装。

```go
import (
    "context"
    "time"
    "github.com/tiamxu/kit/redis"
)

cfg := &redis.Config{
    Address:      "localhost:6379",
    Password:     "",
    DB:           0,
    PoolSize:     20,
    MaxIdle:      15,
    MinIdle:      0,
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
val, err := client.Get(ctx, "key").Result()
if redis.IsNil(err) {
    // key 不存在
}

// 批量操作
values, err := client.MGet(ctx, "key1", "key2", "key3")
client.MSet(ctx, map[string]string{"k1": "v1", "k2": "v2"})

// 安全遍历 key（替代 KEYS 命令）
keys, err := client.Scan(ctx, "prefix:*", 100)

// 原子操作
count, _ := client.Incr(ctx, "counter")
count, _ := client.IncrBy(ctx, "counter", 10)
```

#### 2. Cache 缓存层

面向业务的缓存操作，内部使用 redis.Client 作为存储后端，提供序列化、压缩、分布式锁等功能。

```go
import (
    "context"
    "time"
    "github.com/tiamxu/kit/cache"
    "github.com/tiamxu/kit/redis"
)

redisClient, _ := redis.NewClient(redisCfg)

c := cache.NewRedisCache(redisClient, cache.Options{
    DefaultTTL:      10 * time.Minute,
    NilCacheTTL:     3 * time.Minute,
    EnableCompress:  true,
    CompressMinSize: 2048,
})

// 字符串缓存
c.Set(ctx, "user:1:name", "alice", 0)
name, err := c.Get(ctx, "user:1:name")
if errors.Is(err, cache.ErrNotFound) {
    // 缓存未命中
}

// 对象缓存（自动序列化 + Gzip 压缩）
type UserProfile struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

c.SetObj(ctx, "user:1:profile", &UserProfile{Name: "alice", Age: 30}, 0)

var result UserProfile
found, _ := c.GetObj(ctx, "user:1:profile", &result)

// 批量操作
c.MSet(ctx, map[string]string{"k1": "v1", "k2": "v2"}, 5*time.Minute)
results, _ := c.MGet(ctx, "k1", "k2")

// 原子计数器
count, _ := c.Incr(ctx, "page:views")
count, _ := c.IncrBy(ctx, "score", 10)

// 分布式锁
locked, _ := c.TryLock(ctx, "resource:1", 10*time.Second)
if locked {
    defer c.Unlock(ctx, "resource:1")
    // 执行业务逻辑
}

// TTL 管理
ttl, _ := c.GetTTL(ctx, "key")
c.Expire(ctx, "key", 30*time.Minute)

// 删除和存在检查
c.Delete(ctx, "key1", "key2")
exists, _ := c.Exists(ctx, "key1", "key2")
```

#### 3. 数据库模块

基于 sqlx，支持 MySQL、PostgreSQL、ClickHouse，提供连接池管理和事务。

```go
import "github.com/tiamxu/kit/sql"

cfg := &sql.Config{
    Driver:          "mysql",
    Database:        "test",
    Username:        "root",
    Password:        "password",
    Host:            "localhost",
    Port:            3306,
    MaxIdleConns:    5,
    MaxOpenConns:    10,
    ConnMaxLifetime: 300,
    ConnMaxIdleTime: 60,
}

db, err := sql.Connect(cfg)
if err != nil {
    panic(err)
}

// 查询（sqlx 原生能力）
var users []User
db.Select(&users, "SELECT * FROM users WHERE age > ?", 18)

var user User
db.Get(&user, "SELECT * FROM users WHERE id = ?", 1)

// 增删改
db.Exec("INSERT INTO users (name, age) VALUES (?, ?)", "alice", 30)
db.Exec("UPDATE users SET age = ? WHERE id = ?", 31, 1)
db.Exec("DELETE FROM users WHERE id = ?", 1)

// 判断空结果
err := db.Get(&user, "SELECT * FROM users WHERE id = ?", 999)
if sql.IsNoRows(err) {
    // 没有查到数据
}

// 事务（自动提交/回滚）
err = db.TransactCallback(func(tx *sqlx.Tx) error {
    _, err := tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromID)
    if err != nil {
        return err
    }
    _, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", amount, toID)
    return err
})

// 事务（嵌套，复用外部事务）
err = db.TransactCallback(func(tx *sqlx.Tx) error {
    tx.Exec("UPDATE users SET name = ? WHERE id = ?", "alice", 1)
    return db.TransactCallback(func(tx2 *sqlx.Tx) error {
        tx2.Exec("UPDATE orders SET status = ? WHERE user_id = ?", "active", 1)
        return nil
    }, tx) // 复用同一个事务
})
```

#### 3.1 数据库 + 缓存组合

业务层自行组合 sql 和 cache，实现缓存优先的查询模式。

```go
import (
    "context"
    "fmt"
    "time"
    "github.com/tiamxu/kit/cache"
    "github.com/tiamxu/kit/redis"
    "github.com/tiamxu/kit/sql"
)

// GetUser 缓存优先查询
func GetUser(ctx context.Context, db *sql.DB, c *cache.RedisCache, id int) (*User, error) {
    cacheKey := fmt.Sprintf("user:%d", id)

    // 1. 先查缓存
    var user User
    found, err := c.GetObj(ctx, cacheKey, &user)
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
            // 空值缓存，防止缓存穿透
            c.Set(ctx, cacheKey, "", 3*time.Minute)
            return nil, nil
        }
        return nil, err
    }

    // 3. 写入缓存
    c.SetObj(ctx, cacheKey, &user, 10*time.Minute)
    return &user, nil
}

// UpdateUser 更新数据库 + 删除缓存
func UpdateUser(ctx context.Context, db *sql.DB, c *cache.RedisCache, id int, name string) error {
    err := db.TransactCallback(func(tx *sqlx.Tx) error {
        _, err := tx.Exec("UPDATE users SET name = ? WHERE id = ?", name, id)
        return err
    })
    if err != nil {
        return err
    }

    // 删除缓存，下次查询时重新加载
    cacheKey := fmt.Sprintf("user:%d", id)
    c.Delete(ctx, cacheKey)
    return nil
}
```

#### 4. Kafka 模块

基于 kafka-go，支持生产者和消费者。

```go
import "github.com/tiamxu/kit/kafka"

// 生产者
cfg := &kafka.Config{
    Brokers:      []string{"localhost:9092"},
    Topic:        "test-topic",
    MaxRetries:   3,
    BatchTimeout: 100 * time.Millisecond,
    BatchSize:    100,
}

producer, _ := kafka.NewKafkaProducer(cfg)
defer producer.Close()

producer.SendMessage("test-topic", []byte("key"), []byte("value"))

// 消费者
consumer, _ := kafka.NewKafkaConsumer(
    []string{"localhost:9092"}, "test-topic", "group-id",
)
consumer.ConsumeMessage()
```

#### 5. 日志模块

基于 zap + lumberjack，支持多种输出组合，Kafka 异步写入不阻塞业务。

```go
import "github.com/tiamxu/kit/log"

// 仅控制台输出（默认）
cfg := &log.Config{
    Level:  "info",
    Type:   "stdout",
    Format: "json",
}

// 仅文件输出（自动轮转）
cfg := &log.Config{
    Level:      "info",
    Type:       "file",
    Format:     "json",
    FilePath:   "logs",
    FileName:   "app.log",
    MaxSize:    100,
    MaxBackups: 5,
    MaxAge:     30,
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

// 控制台 + Kafka 同时输出（异步写入，不阻塞业务）
cfg := &log.Config{
    Level:  "info",
    Type:   "stdout+kafka",
    Format: "json",
    Kafka: kafka.Config{
        Brokers: []string{"localhost:9092"},
        Topic:   "app-logs",
    },
}

if err := log.InitLogger(cfg); err != nil {
    panic(err)
}
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

// 带上下文的日志（自动提取 request_id、trace_id）
log.WithContext(ctx).Infof("request processed")
```

#### 6. 向量存储模块

支持 Milvus 和 Qdrant，通过工厂函数统一创建。

```go
import "github.com/tiamxu/kit/vectorstore"

cfg := &vectorstore.VectorStoreConfig{
    Type: "milvus",
    Milvus: vectorstore.MilvusConfig{
        Address:    "localhost:19530",
        Collection: "documents",
        Dimension:  768,
        Index: vectorstore.IndexConfig{
            Type:       "IVF_FLAT",
            MetricType: "COSINE",
            NList:      128,
        },
    },
}

store, err := vectorstore.NewVectorStore(cfg, embedder)
if err != nil {
    panic(err)
}

ctx := context.Background()
store.Initialize(ctx)
store.AddDocuments(ctx, docs)
results, _ := store.Search(ctx, "query text", 5)
store.Close(ctx)
```

#### 7. HTTP 模块

基于 Gin，提供服务器创建、中间件和优雅关闭。

```go
import "github.com/tiamxu/kit/http"

cfg := http.GinServerConfig{
    Address:         ":8080",
    ReadTimeout:     30 * time.Second,
    WriteTimeout:    30 * time.Second,
    AccessLogFormat: http.DefaultAccessLogFormat,
}

router := http.NewGin(cfg)

router.GET("/api/users", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "users": []string{"user1", "user2"},
    })
})

srv, err := http.StartServer(router, cfg)
if err != nil {
    log.Errorf("start server error: %v", err)
    return
}

// 优雅关闭
if err := http.ShutdownServer(srv); err != nil {
    log.Errorf("shutdown server error: %v", err)
}
```

**可用中间件：**

| 中间件 | 说明 |
|--------|------|
| `RequestIDMiddleware()` | 请求 ID 生成（优先读取 X-Request-ID Header） |
| `AccessLogMiddleware(format)` | 访问日志（默认格式按状态码分级输出） |
| `TimeoutMiddleware(timeout)` | 请求超时控制（超时自动返回 408，panic 正确传播） |
| `ErrorHandler()` | 统一错误处理（参数校验、渲染、业务错误分类） |
| `corsMiddleware(config)` | CORS 跨域配置（支持通配符域名匹配） |

**自定义访问日志格式：**

```go
// 使用自定义格式（非默认格式时，按模板字符串输出）
cfg := http.GinServerConfig{
    Address:         ":8080",
    AccessLogFormat: `${time} | ${status} | ${latency} | ${method} ${path}`,
}
// 可用占位符：time, status, latency, client_ip, method, path, request_id,
//            user_agent, error, host, query, real_ip, referer, protocol
```

**统一错误响应：**

```go
// 业务代码中使用 gin.Error 传递错误，ErrorHandler 会自动分类处理
router.GET("/api/users/:id", func(c *gin.Context) {
    user, err := getUser(c.Param("id"))
    if err != nil {
        _ = c.Error(fmt.Errorf("user not found: %w", err)) // 自动返回 404
        return
    }
    c.JSON(http.StatusOK, user)
})
```

## 配置说明

### Redis 客户端配置

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| Address | string | Redis 服务器地址 | :6379 |
| Password | string | Redis 密码 | "" |
| DB | int | Redis 数据库 | 0 |
| PoolSize | int | 连接池最大连接数 | 20 |
| MaxIdle | int | 连接池最大空闲连接数 | 15 |
| MinIdle | int | 连接池最小空闲连接数 | 0 |
| DialTimeout | int | 建立连接超时时间（秒） | 5 |
| ReadTimeout | int | 读取超时时间（秒） | 10 |
| WriteTimeout | int | 写入超时时间（秒） | 10 |

### Cache 缓存配置

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| DefaultTTL | time.Duration | 默认过期时间，0 表示不过期 | 0 |
| NilCacheTTL | time.Duration | 空值缓存 TTL（防缓存穿透），0 表示不缓存空值 | 0 |
| EnableCompress | bool | 是否启用 Gzip 压缩 | false |
| CompressMinSize | int | 压缩阈值（字节），小于此值不压缩 | 2048 |

### 数据库配置

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| Driver | string | 数据库驱动（mysql、postgres、clickhouse） | - |
| Database | string | 数据库名称 | - |
| Username | string | 数据库用户名 | - |
| Password | string | 数据库密码 | - |
| Host | string | 数据库主机 | - |
| Port | int | 数据库端口 | 3306（mysql）、5432（postgres）、9000（clickhouse） |
| MaxIdleConns | int | 最大空闲连接数 | 5 |
| MaxOpenConns | int | 最大打开连接数 | 10 |
| ConnMaxLifetime | int | 连接最大生命周期（秒） | 300 |
| ConnMaxIdleTime | int | 连接最大空闲时间（秒） | 60 |
| ReadTimeout | int | ClickHouse 读取超时（秒） | 10 |
| WriteTimeout | int | ClickHouse 写入超时（秒） | 10 |

### Kafka 配置

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| Brokers | []string | Kafka broker 地址列表 | - |
| Topic | string | 默认主题 | - |
| MaxRetries | int | 最大重试次数 | 3 |
| RetryInterval | time.Duration | 重试间隔 | 100ms |
| BatchTimeout | time.Duration | 批量提交超时 | 100ms |
| BatchSize | int | 批量大小 | 100 |

### 日志配置

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| Level | string | 日志级别（debug、info、warn、error、fatal、panic） | info |
| FilePath | string | 日志文件路径 | logs |
| FileName | string | 日志文件名 | app.log |
| MaxSize | int | 日志文件最大大小（MB） | 100 |
| MaxBackups | int | 最大备份文件数 | 5 |
| MaxAge | int | 日志文件最大保存天数 | 30 |
| Compress | bool | 是否压缩日志文件 | true |
| Type | string | 日志输出类型（stdout、file、kafka、stdout+file、stdout+kafka） | stdout |
| Format | string | 日志格式（text、json），Kafka 端强制 JSON | text |
| Kafka | kafka.Config | Kafka 输出配置（Type 包含 kafka 时生效） | - |

### 向量存储配置

#### Milvus

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| Address | string | Milvus 服务器地址 | - |
| DBName | string | 数据库名称 | "" |
| Collection | string | 集合名称 | - |
| Dimension | int | 向量维度 | 768 |
| MaxLength | int | 文本字段最大长度 | 65535 |
| Username | string | 认证用户名 | "" |
| Password | string | 认证密码 | "" |
| Index.Type | string | 索引类型（IVF_FLAT、IVF_SQ8、HNSW） | - |
| Index.MetricType | string | 度量类型（L2、IP、COSINE、HAMMING、JACCARD） | L2 |
| Index.NList | int | IVF 索引聚类数 | 128 |

#### Qdrant

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| Address | string | Qdrant 服务器地址 | - |
| Collection | string | 集合名称 | - |
| Dimension | int | 向量维度 | 768 |
| Distance | string | 距离度量（Cosine、Euclid、Dot、Manhattan） | Cosine |
| ApiKey | string | API 密钥 | "" |
| Https | bool | 是否启用 TLS | false |

## 许可证

MIT License
