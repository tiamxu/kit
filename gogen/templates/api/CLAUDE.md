# CLAUDE.md

## 执行规范

本项目开发规范以 `AGENTS.md` 为准。
Claude Code 执行任何代码修改前，必须先阅读：

- `AGENTS.md`
- `doc/项目开发规范.md`

## 核心约束

- 默认使用中文回复。
- 修改前先判断任务复杂度。
- 复杂任务必须先出方案，等待确认后再改代码。
- 按 `api -> service -> repo -> model` 分层。
- Go 项目默认使用 `kit/log`、`kit/http`、`kit/sql`。
- 禁止擅自替换日志库、HTTP 框架、数据库访问库或 ORM。
- 禁止硬编码密钥、Token、数据库密码。
