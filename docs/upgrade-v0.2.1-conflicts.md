# v0.2.1 升级冲突记录

日期：2026-09-06
分支：`upgrade/v0.2.1`
上游：官方稳定标签 `v0.2.1`

Git 合并阶段发现的 66 个文本冲突已全部处理，`git diff --name-only --diff-filter=U` 为空。当前分支将创建收敛提交，`dev`、`main` 和生产环境未修改。

## 行为级冲突与解决方案

已按悦享长期规则收敛实现和测试契约，以下项目均已由 `go test ./...` 或前端完整 Vitest 覆盖：

- Astra：只接受精确公开 ID `gpt-6-astra`，拒绝 `gpt-6`、`astra` 和未知后缀；更新归一化、目录和收费测试。
- 流式/Anthropic：首输出前缺终止、截断 JSON、断流和响应头超时统一返回 `UpstreamFailoverError`；首输出后保留内容并发送协议合法失败终止事件。
- previous response：简单模式允许跨分组找回绑定账号，测试夹具开启数据库回退；正式调度仍校验分组、状态、并发、能力和限额。
- WS replay：429 使用当前轮完整 payload 重放；无 `previous_response_id` 时过滤孤立工具输出；无效 encrypted lineage 不进入下一轮；首帧上限为 64 MiB。
- 容量/cyber/429：提前返回前先完成错误码改写、cyber policy 标记和 `x-codex-*` 使用率落库。
- 价格：悦享六个 OpenAI 模型固定价格优先于动态源和通用覆盖；通用覆盖只作用于非悦享模型，Fast、Flex/Batch、Ultrafast、推理和 272K 规则保留。
- 前端：配额监控三模式、账号回填/搜索/解绑、Codex 目录下载、模型广场、Usage 隐藏列迁移、Grok 视频白名单隔离和中英文 Ultrafast 文案均已更新。
- Compose：补齐上游新增网关变量（请求体限制、连接池、调度 outbox、图片流保活等），生产 Compose 也显式传入同一组默认值；容器停止宽限期设为 6 分钟。
- 平滑切换：隔离 Caddy 演练证明现有配置重载时 SSE 可完成但 WebSocket 会断开；配置加入 `stream_close_delay 24h` 后两者均可完成。正式切换必须先以独立端口启动新应用并健康检查，再切换 Caddy，旧容器保留到长连接排空后再停止。

## 验证状态

- 后端代码生成、`go test ./...`、构建：通过。
- Go 1.27 构建的 `golangci-lint v2.13.2`：0 issues。
- Go 1.27 `govulncheck v1.7.0`：代码路径 0 vulnerabilities（依赖中存在但未被调用的条目按工具报告保留）。
- 前端冻结依赖、ESLint、TypeScript、完整 Vitest（262 文件、1879 测试）和生产构建：通过。
- 前端 `pnpm audit --prod`：无 high/critical 漏洞。
- `git diff --check`、Caddy 流式路径禁压缩与 `no-store, no-transform`、密钥模式扫描：通过。
- Docker Compose 脚本与 Caddy 策略检查：通过；生产数据库和运行容器未触碰。
- 隔离数据库升级演练：通过；未执行生产迁移，未修改已应用迁移。

## 发布门槛记录

- 2026-09-06：生产快照 `/opt/sub2api/backups/sub2api-business-20260906T114051Z-v021-rehearsal.dump` 已恢复并通过迁移演练（267 -> 287，旧记录 checksum 与核心数据指纹不变，二次执行无变化）。
- 2026-09-06：当前线上仍运行 `release-v0.1.177.6`，应用、PostgreSQL、Redis healthy，应用重启次数为 0；本轮没有修改生产。
- 分支提交 `c197fe7b4` 的 CI `#34033254051` 和 Security Scan `#34033254024` 已通过。Production image 将由固定发布标签触发；当前生产 Compose 是单应用实例，必须使用 canary + Caddy 延迟关闭流程，`docker compose up` 重建不满足平滑切换要求。
