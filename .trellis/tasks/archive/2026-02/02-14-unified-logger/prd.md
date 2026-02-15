# 统一化日志管理 - logrus命名空间化logger

## Goal

将项目中散乱的日志输出（`log.Printf`、`fmt.Printf`、裸 `common.Logger`）统一为基于 logrus 的命名空间化 logger，实现分级、分命名空间、高度可控的全局唯一日志系统。

## Requirements

1. 改造 `core/common/logger.go`，提供 `NamespacedLogger` 类型，支持：
   - 命名空间链式派生：`logger.With("rdb").With("sqlite")` → `[rdb.sqlite]`
   - 标准 logrus 分级：Debug/Info/Warn/Error/Fatal
   - 全局唯一 Logger 单例 + 子 logger 工厂
   - 全局 level 控制（`SetLogLevel`）
2. 替换 `core/tenant_manager.go` 中的 `log.Printf` → 命名空间 logger
3. 替换 `core/rdb/exec_core.go` 中的 `fmt.Printf("[WARN]"...)` → 命名空间 logger
4. 替换 `core/rdb/url_parser.go` 中的 `fmt.Println` → 命名空间 logger
5. 为已有的 `common.Logger` 调用点添加命名空间上下文（rdb、kv、s3、gateway 等）
6. `cmd/` 下的 `fmt.Print*` 是 CLI 用户交互输出，不改动

## Acceptance Criteria

- [ ] `core/common/logger.go` 导出 `NamespacedLogger` 类型和 `NewLogger(namespace)` 工厂
- [ ] 全局 `Logger` 保持单例，`SetLogLevel` 全局生效
- [ ] 所有 `core/` 下的 `log.Printf` 和 `fmt.Printf` 日志调用已替换
- [ ] 每个模块使用自己的命名空间 logger（如 `rdb`、`rdb.sqlite`、`kv`、`s3`、`tenant`）
- [ ] 日志输出格式包含命名空间标识
- [ ] `go build ./...` 编译通过

## Technical Notes

- logrus 已在 go.mod 中（v1.9.4）
- 现有 `ErrorBuilder` 的命名空间模式可参考但不需要耦合
- 使用 `logrus.Entry.WithField("ns", namespace)` 实现命名空间
- CLI 输出（cmd/）保持 fmt 不变

## Out of Scope

- JSON formatter / 文件输出（后续可扩展）
- cmd/ 下的 CLI 交互输出
- 日志轮转
