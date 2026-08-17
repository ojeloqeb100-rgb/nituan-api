# 视频模型按秒及分辨率计费改造总结与审核说明

> 文档日期：2026-08-16
> 目标代码库：New API v1.0.0-rc.24 本地源码
> 本地路径：`D:\API`
> 当前状态：改造保存在本地工作区，尚未提交、推送或部署到生产环境
> 审核用途：供 Cursor 对实现正确性、计费安全性和回归风险进行独立审核

## 1. 需求背景

目标模型以 `doubao-seedance-2.0-mini` 为例，上游提供 OpenAI 兼容视频接口：

- 提交端点：`POST /v1/videos`
- 请求字段：`model`、`prompt`、`seconds`、`size`
- 返回方式：异步任务，提交后返回任务 ID，再通过轮询获取状态和结果

最终确认的计费需求：

1. 不按 Token 计费。
2. 不考虑参考图片数量。
3. 根据用户请求中的 `seconds` 计费。
4. 根据用户请求中的 `size` 区分 `480P`、`720P`、`1080P` 三档价格。
5. 每个视频模型的三档单价由管理员在 New API 后台独立配置。
6. 同一分辨率的横屏和竖屏写法归入同一个价格档位。

本次实现的计费口径是“请求时长”，不是任务完成后解析视频文件得到的真实时长：

```text
应扣美元金额 = 对应分辨率每秒单价 × 请求 seconds × 用户组倍率
应扣额度 = 应扣美元金额 × QuotaPerUnit
```

例如配置 `480P=$0.5/s`、`720P=$0.8/s`、`1080P=$1.2/s`，用户组倍率为 `1`，请求 `10` 秒时：

| 请求分辨率 | 美元金额 | 转换前公式 |
|---|---:|---|
| 480P | $5 | `0.5 × 10` |
| 720P | $8 | `0.8 × 10` |
| 1080P | $12 | `1.2 × 10` |

## 2. 方案结论

本次没有使用现有“按表达式”模式，而是新增独立的 `video` 计费模式，原因如下：

- 现有表达式变量面向 Token、图像 Token 和音频 Token，不直接提供视频请求的 `seconds`、`size` 计费契约。
- 视频任务是异步链路，需要在任务提交、预扣、任务持久化、轮询完成和失败退款之间保持同一份计费快照。
- 视频每秒单价通常很小，若先转整数额度再乘秒数，可能因提前取整导致少扣或零扣。本实现先计算完整浮点金额，再进行一次额度转换。
- 三档价格需要在后台可视化维护，并在公共模型价格页展示，不适合隐藏在固定代码或复杂表达式中。

新增配置项：

```json
{
  "billing_setting.billing_mode": {
    "doubao-seedance-2.0-mini": "video"
  },
  "billing_setting.video_prices": {
    "doubao-seedance-2.0-mini": {
      "480p": 0.5,
      "720p": 0.8,
      "1080p": 1.2
    }
  }
}
```

实际写入系统设置时，上述两个配置值分别以 JSON 字符串保存，与现有 `billing_setting` 配置方式一致。

## 3. 管理后台使用方式

预期操作路径：

```text
系统管理 -> 计费与支付 -> 模型定价 -> 选择目标模型 -> 按秒
```

管理员需要填写：

- 480P 每秒美元单价
- 720P 每秒美元单价
- 1080P 每秒美元单价

保存约束：

- 三档价格必须全部存在。
- 每档价格必须是有限数字且大于 `0`。
- 不允许额外的未知档位，例如 `2k`。
- 模型名不能为空。
- 模型设为 `video` 模式时必须存在完整的视频价格配置。

公共模型价格页会显示：

- 计费模式徽标“按秒”。
- 480P、720P、1080P 三档每秒价格。
- 按用户组倍率换算后的三档价格。
- 充值倍率和美元汇率开启时的对应展示价格。

为避免上游倍率同步覆盖本地视频绝对价格，倍率同步逻辑会跳过 `video` 计费模式。

## 4. 请求参数与分辨率映射

### 4.1 `seconds`

- JSON 请求支持整数：`"seconds": 10`。
- JSON 请求支持数字字符串：`"seconds": "10"`。
- multipart 请求从表单字段 `seconds` 读取。
- 显式传入的有效范围为 `1..3600` 秒。
- 未提供有效时长时沿用原任务链路的默认值 `4` 秒。
- 兼容原有 `duration` 字段；公共任务校验会限制其最大值。

### 4.2 `size`

- 未提供 `size` 时默认使用 `720x1280`，即 720P 档。
- 大小写不敏感。
- 会去除空格。
- 分隔符兼容 `x`、`X`、`*`、`×`。
- 只接受明确映射，不使用像素范围模糊匹配，避免未知尺寸误落入低价档。

| 计费档位 | 当前支持的规范化写法 |
|---|---|
| 480P | `480p`、`854x480`、`480x854`、`832x480`、`480x832`、`640x480`、`480x640` |
| 720P | `720p`、`1280x720`、`720x1280` |
| 1080P | `1080p`、`1920x1080`、`1080x1920`、`1792x1024`、`1024x1792` |

当前明确拒绝：

- `768p`
- `2k`
- `sd`
- `hd`
- `fhd`
- `2048x1080`
- 其他未列入映射表的尺寸

这些写法的实际语义在不同上游间可能不一致，直接归档可能造成少扣费，因此目前采取失败关闭策略，返回参数错误。

## 5. 异步计费调用链

```text
客户端 POST /v1/videos
  -> 解析并校验 model / seconds / size
  -> 将 size 规范化为 480p / 720p / 1080p
  -> 按原始模型名读取该档每秒单价
  -> 单价 × seconds × 用户组倍率
  -> 一次性转换为 quota 并预扣
  -> 请求上游 /v1/videos
  -> 返回公开 task_xxxx，保存上游真实任务 ID
  -> 保存 BillingContext 计费快照
  -> 轮询异步任务
       -> 成功：保持提交时按秒计费结果，不再按 Token 重算
       -> 失败：走异步任务统一退款逻辑
```

关键行为：

1. 只有任务首次提交时预扣；渠道重试复用已有 BillingSession，避免重复预扣。
2. 提交阶段发生错误时，由控制器的 `defer Refund` 退回预扣。
3. 上游已接受但异步任务最终失败时，复用现有 `RefundTaskQuota` 退款流程。
4. `PerCallBilling=true` 时，轮询成功阶段跳过 adaptor 调整和 Token 差额结算，避免二次扣费或退款。
5. 任务保存 `BillingMode`、`VideoResolution`、`ModelPrice`、`OtherRatios` 等快照。
6. Remix 优先复用原任务计费快照，避免管理员改价后重解释历史任务价格。
7. 旧任务没有新快照时保留兼容回退逻辑，并对历史时长做上限保护。

## 6. 后端改造说明

### 6.1 视频价格配置与校验

新增文件：

- `setting/billing_setting/video_billing.go`
- `setting/billing_setting/video_billing_test.go`

职责：

- 定义三档分辨率常量和价格结构。
- 维护精确尺寸别名映射。
- 提供分辨率规范化和模型视频价格读取函数。
- 校验视频价格 JSON、计费模式与价格配置的完整性。

相关修改：

- `setting/billing_setting/tiered_billing.go`：增加 `BillingModeVideo` 与 `VideoPrices` 配置字段。
- `model/option.go`：保存配置前调用视频计费校验。

### 6.2 请求解析与 Sora/OpenAI Video 适配

- `relay/common/relay_info.go`：`seconds` 同时支持 JSON 数字和字符串；任务上下文增加视频计费字段。
- `relay/channel/task/sora/adaptor.go`：校验时长和尺寸、计算每秒单价、支持 JSON/multipart 请求体、保留模型映射和异步响应处理。
- `relay/channel/adapter.go`：新增 `TaskPerCallPriceEstimator` 接口，由需要绝对按次价格的任务适配器实现。

当前只有实现了该估价接口的任务适配器才能使用 `video` 模式；本次目标是 OpenAI 兼容 `/v1/videos` 的 Sora 任务适配器，不等于所有 New API 视频渠道已自动支持。

### 6.3 价格计算与异步任务快照

- `relay/helper/price.go`：增加“绝对价格 + 所有倍率后一次转换 quota”的计算入口，防止小数单价提前截断。
- `relay/relay_task.go`：识别 `video` 模式、调用适配器获取分辨率单价、合并秒数倍率、预扣并维护 Remix 快照。
- `controller/relay.go`：任务持久化时保存 `BillingMode` 和 `VideoResolution`。
- `model/task.go`：扩展 `TaskBillingContext`。
- `controller/ratio_sync.go`：倍率同步跳过本地 `video` 绝对价格配置。

### 6.4 公共价格数据

- `model/pricing.go`：公共 Pricing 数据增加 `video_prices`，`video` 模式按固定价格类型对外展示。
- `relay/helper/price.go`：模型是否已定价的判断支持 `video` 模式。

## 7. 前端改造说明

模型定价后台：

- 新增“按秒”模式。
- 新增 480P、720P、1080P 三个美元/秒输入框。
- 编辑、快照对比、筛选、保存、删除模型配置时同步维护 `video_prices`。
- 防止视频模式被普通模型倍率数据覆盖。

公共价格页：

- 模型卡片展示三档每秒价格。
- 表格价格列展示三档价格。
- 模型详情展示基础价格及各用户组价格。
- 计费模式徽标显示视频按秒计费。

国际化：

- 已同步 `en`、`zh`、`zh-TW`、`fr`、`ja`、`ru`、`vi` 七种语言中的视频计费文案。

## 8. 测试与验证记录

已覆盖的后端单元测试方向：

- 分辨率别名规范化及未知尺寸拒绝。
- `seconds` 数字和字符串解析。
- `seconds` 的零值、负数、非数字和超上限校验。
- 三档价格完整性、零价格、未知档位和非法 JSON 校验。
- `video` 模式必须存在完整价格。
- 按秒倍率写入 PriceData。
- 绝对每秒价格在乘时长前不发生整数截断。
- Remix 复用历史视频单价快照。
- 视频价格配置被识别为有效模型定价。

已有验证记录：

| 验证项 | 结果 | 说明 |
|---|---|---|
| 前端 `tsgo -b` | 通过 | TypeScript 类型检查通过 |
| 前端 `rsbuild build` | 通过 | 前端生产构建通过 |
| 受影响 TS/TSX `oxlint` | 通过 | 受影响文件未发现新增 lint 错误 |
| `git diff --check` | 通过 | 未发现空白符错误 |
| 后端 `go test ./...` | 前一轮通过 | 当前 PowerShell 会话未找到 Go 命令，未再次复跑 |
| `gofmt` | 前一轮完成 | 当前业务代码未在本文档任务中修改 |
| 真实 Doubao 上游联调 | 未执行 | 仍需使用真实渠道验证提交、轮询、成功与失败路径 |

## 9. 当前已知边界和残余风险

Cursor 审核时应将以下内容视为重点，而不是默认已证明正确：

1. **按请求时长计费**：当前不会在任务完成后读取实际视频时长重新结算。若上游实际生成时长与请求 `seconds` 不一致，仍按请求值收费。
2. **适配器范围**：当前核心实现落在 Sora/OpenAI Video 任务适配器；其他视频渠道即使也有 `seconds`、`size` 字段，仍需确认是否进入同一适配器。
3. **上游状态值**：需要用真实 Doubao 上游确认完成状态使用 `completed`、`succeeded` 还是其他值，并核对现有状态映射。
4. **配置保存顺序**：`billing_mode` 与 `video_prices` 会互相校验，需审核后台批量保存时的顺序和原子性，确保首次从普通模式切换到视频模式不会因读取旧配置而失败。
5. **multipart 非数字时长**：需重点审核 multipart 解析对非法 `seconds` 字符串是否会静默回落为默认 4 秒；计费字段不应容忍歧义输入。
6. **模型映射**：应确认计费始终使用用户请求的原始模型名读取本地价格，而上游请求使用映射后的模型名，不能被渠道映射绕过定价。
7. **Remix 语义**：当前优先复用历史快照。需要确认业务期望是沿用原视频价格与时长，还是应按新的 Remix 请求参数重新计费。
8. **退款幂等**：异步失败退款依赖现有任务状态 CAS 和 `task.Quota` 清零机制，需审核并发轮询与重复失败通知不会重复退款。
9. **日志可审计性**：需确认消费日志和 Task Logs 能看到模型、分辨率单价、seconds、组倍率和最终 quota，便于人工对账。
10. **未识别尺寸**：目前采取拒绝请求，不会自动映射 `768p`、`2k`、`hd` 等写法。新增上游时必须先确认官方尺寸语义再扩展映射。
11. **真实环境差异**：本地源码尚未与生产数据库配置、Redis、任务轮询调度及真实上游做完整联调。

## 10. Cursor 审核清单

建议 Cursor 按以下顺序进行只读审核，先给出问题清单，不要直接修改代码：

### 10.1 计费安全

- [ ] 验证公式确实是 `分辨率单价 × seconds × groupRatio × QuotaPerUnit`，且只转换一次整数 quota。
- [ ] 验证 `seconds` 的所有输入路径都限制在 `1..3600`，缺省值行为一致。
- [ ] 验证未知 `size` 一律失败，不会落入低价档或免费路径。
- [ ] 验证缺少模型价格、缺少任一档价格、零价格、NaN、Inf 时都拒绝计费配置或请求。
- [ ] 验证自动组切换、特殊组倍率和渠道重试不会漏乘倍率或重复扣费。
- [ ] 验证额度溢出时失败或饱和，不会变成负数后反向加余额。

### 10.2 异步生命周期

- [ ] 验证提交失败会退回本次预扣。
- [ ] 验证上游接受后任务最终失败会退款，且重复轮询不会重复退款。
- [ ] 验证任务成功后不会再按 Token 或 adaptor 结果二次结算。
- [ ] 验证公开 task ID 与上游真实 task ID 分离，不影响轮询和退款定位。
- [ ] 验证 Remix 新旧任务两条兼容路径都不会改变历史收费或产生零价。

### 10.3 配置与前端

- [ ] 验证首次新增视频模型、编辑价格、切换计费模式、删除模型配置均能正确保存。
- [ ] 验证 `billing_mode` 和 `video_prices` 保存顺序不会触发交叉校验死锁。
- [ ] 验证三档价格在后台列表、编辑弹窗、公共价格页、分组价格页显示一致。
- [ ] 验证七种语言不存在缺失 key、直接展示英文 key 或 JSON 重复 key。
- [ ] 验证倍率同步、模型扫描或默认倍率逻辑不会覆盖 `video` 模式。

### 10.4 兼容与回归

- [ ] 验证普通按 Token、按次和按表达式模型行为不变。
- [ ] 验证原有 Sora 视频固定价格和 1080P `size` 倍率逻辑不被破坏。
- [ ] 验证 JSON、multipart、模型映射、渠道重试和请求体重放均正常。
- [ ] 验证公共 Pricing API 新字段对旧前端和其他调用方保持向后兼容。

## 11. Cursor 建议执行的验证命令

```powershell
Set-Location D:\API

# 先查看本次语义差异；Dockerfile.dev 是用户原有修改，不属于视频计费改造
git diff -- . ':(exclude)Dockerfile.dev' ':(exclude)docs/video-billing-implementation-review.md'

# 后端
go test ./...

# 前端
pnpm --dir web typecheck
pnpm --dir web build
pnpm --dir web lint

# 通用检查
git diff --check
git status --short
```

真实渠道至少需要覆盖以下用例：

| 用例 | 请求 | 预期 |
|---|---|---|
| 480P | `seconds=5,size=854x480` | 按 480P 单价扣 5 秒 |
| 720P 横屏 | `seconds=10,size=1280x720` | 按 720P 单价扣 10 秒 |
| 720P 竖屏 | `seconds="10",size=720x1280` | 与横屏同价 |
| 1080P | `seconds=4,size=1920x1080` | 按 1080P 单价扣 4 秒 |
| 默认值 | 不传 `seconds`、`size` | 按 720P、4 秒处理 |
| 非法时长 | `seconds=0/-1/3601/abc` | 请求失败且不扣费 |
| 非法尺寸 | `size=2k/768p/hd` | 请求失败且不扣费 |
| 渠道重试 | 首渠道失败、次渠道成功 | 只预扣和结算一次 |
| 异步失败 | 提交成功后任务失败 | 预扣全额退回且退款仅一次 |
| 异步成功 | 提交后任务成功 | 保持预扣金额，不按 Token 二次结算 |
| Remix | 基于历史任务 remix | 价格快照行为符合确认后的业务口径 |

## 12. 改动文件范围

后端与测试：

```text
controller/ratio_sync.go
controller/relay.go
model/option.go
model/pricing.go
model/task.go
relay/channel/adapter.go
relay/channel/task/sora/adaptor.go
relay/channel/task/sora/adaptor_test.go
relay/common/relay_info.go
relay/common/relay_info_test.go
relay/helper/price.go
relay/helper/price_test.go
relay/relay_task.go
setting/billing_setting/tiered_billing.go
setting/billing_setting/video_billing.go
setting/billing_setting/video_billing_test.go
```

前端与国际化：

```text
web/src/features/models/components/drawers/model-mutate-drawer.tsx
web/src/features/pricing/components/model-billing-mode-badge.tsx
web/src/features/pricing/components/model-card.tsx
web/src/features/pricing/components/model-details.tsx
web/src/features/pricing/components/pricing-columns.tsx
web/src/features/pricing/lib/price.ts
web/src/features/pricing/types.ts
web/src/features/system-settings/billing/index.tsx
web/src/features/system-settings/billing/section-registry.tsx
web/src/features/system-settings/models/index.tsx
web/src/features/system-settings/models/model-pricing-core.ts
web/src/features/system-settings/models/model-pricing-sheet.tsx
web/src/features/system-settings/models/model-pricing-snapshots.ts
web/src/features/system-settings/models/model-ratio-form.tsx
web/src/features/system-settings/models/model-ratio-table-columns.tsx
web/src/features/system-settings/models/model-ratio-visual-editor.tsx
web/src/features/system-settings/models/ratio-settings-card.tsx
web/src/features/system-settings/types.ts
web/src/i18n/locales/en.json
web/src/i18n/locales/fr.json
web/src/i18n/locales/ja.json
web/src/i18n/locales/ru.json
web/src/i18n/locales/vi.json
web/src/i18n/locales/zh-TW.json
web/src/i18n/locales/zh.json
```

工作区中的 `Dockerfile.dev` 修改在本任务开始前已存在，不属于本次视频计费改造，审核时不要误算或回退。

## 13. 改动规模快照

以下统计是在创建本文档前得到的业务代码快照，不包含本文档本身：

- 视频计费相关文件：41 个，其中 39 个已跟踪修改、2 个新文件。
- 新增：1331 行。
- 删除：87 行。
- 净增加：1244 行。

统计包含测试和国际化文案；不包含用户原有 `Dockerfile.dev` 的新增 4 行、删除 1 行。

## 14. 审核输出要求

建议 Cursor 最终按严重度输出：

1. `P0`：可导致少扣、零扣、重复退款、余额反增、越权或生产不可用。
2. `P1`：主流程错误、配置无法保存、异步任务无法完成或价格展示严重不一致。
3. `P2`：边界兼容、测试缺口、可观测性或维护性问题。
4. 若没有发现问题，应明确写“未发现阻断问题”，并列出仍未执行的真实上游联调和生产环境风险。

本文档仅总结当前实现和验证证据，不代表代码已经通过独立审核或具备直接上线条件。
