# gogen

`gogen` 是 kit 仓库内置的 Go 项目脚手架命令，用于生成基于 `github.com/tiamxu/kit` 的 API 项目骨架。

当前仅支持生成 API 项目：

```bash
go run ./gogen new api --name demo-api --module github.com/tiamxu/demo-api --target ./demo-api
```

## 常用命令

生成基础 API 项目：

```bash
go run ./gogen new api --name demo-api --module github.com/tiamxu/demo-api --target ./demo-api
```

生成带数据库初始化的 API 项目：

```bash
go run ./gogen new api --name demo-api --module github.com/tiamxu/demo-api --target ./demo-api --with-db
```

预览将生成的文件：

```bash
go run ./gogen new api --name demo-api --module github.com/tiamxu/demo-api --target ./demo-api --dry-run
```

本地联调当前 kit 仓库：

```bash
go run ./gogen new api --name demo-api --module github.com/tiamxu/demo-api --target ./demo-api --local-kit-replace
```

跳过依赖整理和测试：

```bash
go run ./gogen new api --name demo-api --module github.com/tiamxu/demo-api --target ./demo-api --skip-tidy --skip-test
```

## 参数说明

### 基础参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--name` | 是 | 项目名称，只能是名称，不能是路径 |
| `--module` | 是 | Go module 路径，例如 `github.com/tiamxu/demo-api` |
| `--target` | 是 | 生成目标目录 |

### 能力参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--with-db` | 否 | 生成数据库配置和 `repo.Init` 初始化代码；默认不生成数据库相关内容 |

### 开发调试参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--dry-run` | 否 | 仅打印将生成的文件路径和 `.gogen/manifest.json`，不写入文件 |
| `--force` | 否 | 允许覆盖非空目标目录中的同名文件 |
| `--skip-tidy` | 否 | 跳过生成后的 `go mod tidy` |
| `--skip-test` | 否 | 跳过生成后的 `go test ./...` |
| `--local-kit-replace` | 否 | 在生成项目的 `go.mod` 中写入本地 kit 仓库 `replace`，用于本地联调；找不到本地 kit 仓库时会报错 |

## 生成内容

生成的 API 项目包含：

```text
api/                 接口响应封装
config/              配置加载和 YAML 配置文件
routes/              路由注册
pkg/e/               错误码
doc/                 项目文档
repo/                数据库初始化入口，仅 `--with-db` 时生成
AGENTS.md            工程执行规范
CLAUDE.md            Claude Code 执行规范
README.md            生成项目说明
go.mod               Go module 文件
main.go              应用入口
```

## 模板规则

模板文件位于 `templates/api/`，目录结构对应生成项目结构。

- `.tmpl` 文件会使用 Go `text/template` 渲染，并在生成时去掉 `.tmpl` 后缀。
- 普通文件会按原内容直接复制。
- 当前模板变量包括 `ProjectName`、`Module`、`KitReplace`、`WithDB`。
- 默认健康检查接口使用 `api.Success` 统一响应，响应消息来自 `pkg/e`。
- 默认不生成数据库配置和 `repo.Init` 初始化代码；传入 `--with-db` 时才生成。

## 生成清单

生成完成后会写入 `.gogen/manifest.json`，记录本次生成的项目类型、项目名称、module 和文件列表。

该文件用于追踪脚手架生成范围，便于后续检查哪些文件来自 `gogen`。当前清单只做记录，不会自动删除、合并或重写业务代码。

## 数据库初始化

默认生成的 API 项目不包含数据库配置，也不会生成 `repo/init.go` 或启动阶段数据库初始化代码。因此新项目可以直接启动 HTTP 服务和 `/api/v1/health`，不依赖本地数据库。

需要数据库时，生成项目时显式传入 `--with-db`：

```bash
go run ./gogen new api --name demo-api --module github.com/tiamxu/demo-api --target ./demo-api --with-db
```

此时会生成 `repo/init.go`，并在 `config/config.yaml` 中写入数据库连接配置：

```yaml
db:
  driver: mysql
  database: app
  username: root
  password: ""
  host: 127.0.0.1
  port: 3306
```

## 安全边界

- 默认不写入本地 `replace github.com/tiamxu/kit => ...`，避免生成项目携带本机路径。
- 只有显式传入 `--local-kit-replace` 时才写入本地 `replace`。
- 显式传入 `--local-kit-replace` 但找不到本地 kit 仓库时会报错，避免静默生成未联调的 `go.mod`。
- 即使传入 `--force`，也不允许将目标目录设置为 kit 仓库根目录。
