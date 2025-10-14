# balance-bot

一个用于周期性检测 EVM 链（当前以 BSC 为主）上地址原生代币余额，并在余额超出配置阈值时通过多种 webhook（企业微信 / 飞书(Lark) / Telegram）发送告警的轻量级守护程序。

## 项目结构（简要）

- `main.go`：程序入口，读取配置并循环检查所有配置的地址余额，触发告警。
- `internal/config/config.go`：配置结构体及 `config.json` 示例的读写。
- `internal/core/evm.go`：通过 RPC（`eth_getBalance`）查询原生代币余额，并包含 BSC RPC 列表与轮询逻辑。
- `internal/utils/*.go`：工具集合，包括 HTTP 客户端、日志封装、消息发送（WeCom / Lark / Telegram）、数值/字符串/hex 工具等。
- `go.mod`：依赖声明（`fasthttp`、`file-rotatelogs` 等）。

## 主要功能

- 周期性按 `config.json` 中的 `interval`（秒）检查每个地址的原生代币余额。
- 当余额低于 `min` 或高于 `max` 时发送告警消息。
- 支持通过企业微信、飞书（Lark）、Telegram webhook 发送告警。
- 自动生成 `config.json` 示例（如果项目目录下没有该文件）。
- 日志系统：使用 `slog` + 按天轮转的文件写入（`logs/` 目录）。

## 配置（config.json）

程序使用项目根目录下的 `config.json`。如果不存在，程序会写入一个示例文件并退出一次，用户需编辑后再次运行。

示例内容：

```json
{
  "webhook": {
    "wecom": "",
    "lark": "",
    "telegram_token": "",
    "telegram_chat_id": ""
  },
  "interval": 30,
  "tokens": [
    {
      "address": "0x1234567890abcdef1234567890abcdef12345678",
      "chainId": "56",
      "name": "MyWallet01",
      "min": 0.1,
      "max": 1000
    }
  ]
}
```

字段说明：

- `webhook.wecom`：企业微信机器人 webhook URL（可选）。
- `webhook.lark`：飞书(Lark) 机器人 webhook URL（可选）。
- `webhook.telegram_token`：Telegram Bot token（可选）。
- `webhook.telegram_chat_id`：Telegram chat id（可选）。
- `interval`：检测间隔（秒），默认 30 秒。
- `tokens`：要监控的地址列表：
  - `address`（必填）：钱包地址。
  - `chainId`（可选）：链 ID（默认 `56`，即 BSC）。
  - `name`（可选）：地址别名，用于通知展示。
  - `min`（可选）：低于该值发送告警（以原生代币为单位，如 BNB/ETH）。默认 `0.1`（见源码默认值）。
  - `max`（可选）：高于该值发送告警（默认不限制）。

注意：当前只查询原生链币（如 BNB/ETH）的余额，不包含 ERC20/ERC721 等代币的余额查询。

## 日志与运行时

- 默认日志目录：`logs/`，按天轮转，同时输出到 stdout。
- 若想自定义日志级别或轮转策略，参考 `internal/utils/logger.go` 中 `InitLogger` / `NewEnhancedLogger` 的参数。

## 通知模板（示例）

- 当余额低于 `min`，通知示例：

  "⚠️ Balance for 0x1234**5678(MyWallet01) on chain 56 is below minimum 0.100000: 0.050000"

- 当余额高于 `max`，类似格式。

## 实现细节（简要）

- 地址余额通过 JSON-RPC `eth_getBalance` 获取，默认 decimals=18（源码中用于将 wei 转为浮点数）。
- BSC RPC 列表位于 `internal/core/evm.go` 的 `BSC_RPC`，使用轮询索引以分散请求压力。
- HTTP 请求使用 `fasthttp` 客户端封装；SendPost/SendGet 均有统一处理与 JSON 编解码。

## 已知限制 / 注意事项

- 仅支持原生代币余额查询，若需 ERC20 代币支持需扩展：
  - 查询合约 `balanceOf`，并处理 token 的 decimals 值。
- 当前只为链 ID `56` 提供 RPC 列表，添加其它链需要在 `GetRPC` 中扩展并提供 RPC 列表。
- 大量地址或非常短的间隔可能需要调整 HTTP 客户端连接数与轮询策略以避免 RPC 被限流。

## 可选扩展

- 支持 ERC20 token 余额与 token 配置（合约地址 + decimals）。
- RPC 健康检查（失败次数达到阈值时从池中剔除并回退到备用 RPC）。
- 告警去重与严重等级（避免连续重复通知）。

## 许可证

请参阅仓库根目录的 `LICENSE` 文件。
