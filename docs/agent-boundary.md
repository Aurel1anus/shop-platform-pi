# pi 插件式接入边界

项目根目录使用 pi 1.4.0（`@earendil-works/pi-coding-agent`），资源位于 `.pi/extensions` 和 `.pi/skills`。旧 `agent/src` 目录已移除。Go 业务服务不受影响。

## 当前结构

```text
shop-platform/
  package.json / package-lock.json   项目本地 pi 依赖
  .pi/
    settings.json                    默认模型、可用工具与重试设置
    SYSTEM.md                        项目系统提示词
    extensions/
      status.ts                      已实现：shop_status 接入状态工具
      commerce/
        index.ts                     可加载的插件入口，业务尚未注册
        catalog.ts                   空占位
        cart.ts                      空占位
        order.ts                     空占位
        knowledge.ts                 空占位
      harness/
        index.ts                     可加载的插件入口，事件钩子待实现
        context.ts / policy.ts       空占位
        events.ts / run.ts / store.ts 空占位
    skills/
      shopping/SKILL.md               导购技能占位说明
      orders/SKILL.md                 购物车/订单技能占位说明
      knowledge/SKILL.md              知识检索技能占位说明
    runtime/                         本地运行缓存，Git 忽略
  src/
    pi-session.mjs                   已实现：创建 SDK 会话并加载本项目资源
    server.ts                        空占位：HTTP/SSE 入口
    clients/commerce.ts               空占位：Go API 客户端
    retrieval/                       空占位：ingest.ts、search.ts、types.ts
  scripts/pi.mjs                     已实现：单次对话与离线接入检查
```

pi 的扩展文件必须导出合法入口，技能文件必须有有效的 name 和 description。因此插件 index.ts 是可加载的无操作函数，SKILL.md 是说明当前边界的有效技能；不会用空模块冒充已实现业务。多文件插件内部的业务文件仍为零字节。

## 安装和启动

要求 Node.js 22.19+。在 shop-platform 根目录执行：

```bash
npm ci
npm run check
npm run pi
```

`npm run pi` 使用项目本地 pi CLI，按 pi 自身的项目资源信任流程加载当前目录的 `.pi`；模型凭证和交互会话遵循用户 pi 配置，默认位于 `~/.pi/agent`。首次进入项目时按 CLI 提示确认是否信任本项目资源。

单次 SDK 对话：

```bash
npm run chat -- "调用 shop_status 工具，说明当前哪些能力已接入。"
```

SDK 入口只加载本项目资源，不加载个人扩展、个人技能和祖先 AGENTS 文件；使用进程内会话，不保存业务聊天历史。它显式信任本仓库 `.pi` 资源，这项配置针对本地开发入口，不是面向任意目录的信任开关。

默认模型为 `deepseek/deepseek-flash`，对应本机已有的 pi 设置。凭证通过 pi 用户配置或供应商环境变量提供，不复制进项目。没有凭证时先在 pi CLI 中 `/login` 配置；不要把密钥写进 SYSTEM.md 或 settings.json。

SDK 单次调用可临时选择模型：

```bash
PI_PROVIDER=deepseek PI_MODEL=deepseek-flash npm run chat -- "你好"
```

交互 CLI 用 pi 原生 `--provider` / `--model` 参数。SDK 对话超时为 60 秒，SDK 会话结束后释放；`check` 不请求模型 API，不要求后端或模型凭证。

默认可用工具为 `read` 和 `shop_status`，没有启用 bash、edit、write。read 用于读取技能资料。当前入口是本地开发工具，尚不是具备用户鉴权和文件访问隔离的公网聊天服务。

## 实现与占位边界

| 部分 | 状态 |
|---|---|
| 项目本地 pi 安装、CLI、SDK 会话 | 已配置 |
| .pi 扩展和技能加载、系统提示词 | 已配置 |
| shop_status 工具 | 已实现，仅报告接入状态，不探测后端 |
| 商品、购物车、订单工具 | 占位 |
| 记忆、权限策略、业务会话/run/trace | 占位 |
| 自写 RAG | 占位 |
| HTTP/SSE 和客户端聊天接入 | 占位，Go 相关接口仍返回 501 |

`COMMERCE_API_BASE_URL` 默认 `http://127.0.0.1:8081/api/v1`，当前仅由 shop_status 展示，尚未用于业务调用。修改该地址不代表业务工具已经接通。

## 数据和业务边界

- Go：账号、商品、库存、购物车、订单、优惠券、评价、文件和原始知识资料。
- harness 扩展：后续实现会话上下文、商品指代、权限、预算、取消和业务持久化。
- RAG：后续实现原文切块、embedding、检索索引和重排，通过插件注册工具。
- skill：工作流指引；不替代业务工具，也不自行提供权限或交易保证。

未来工具通过 clients/commerce.ts 调用业务 API，不直接写业务表。用户身份由已认证的请求上下文注入，不向模型开放任意 account_id，不共用管理员凭证代办交易。写工具仍需业务幂等，取消生成不会撤销已提交订单。

旧聊天、Prompt、trace 和评测 API 当前保留 501。复用原客户端时需要后续增加 HTTP/SSE 适配，并明确唯一的会话/run 数据拥有者。

## 知识与评测

Go 的 documents 管理接口保存和返回原始 content、metadata、来源与 hash；RAG 尚未切块或建索引。商品原文和评测输入在 quality/data 中。

旧 300 条检索用例的 chunk_id 对应旧切块，新实现应映射到来源或商品 ID。Agent 场景评测应检查真实业务结果，不能把占位插件成功加载当成业务通过。

## 验证记录

2026-10-26，在 Node.js 22.23.4 环境验证：

- `npm run check` 通过：真实 SDK 初始化、3 个扩展、3 个技能及状态工具执行；无模型 API 请求。
- `npm run pi -- --version` 返回项目本地版本 `1.4.0`。
- `npm run chat -- "只调用一次 shop_status 工具，然后用一句中文说明当前接入状态。"` 通过：使用已有 DeepSeek 凭证，模型实际调用 shop_status，并根据工具结果生成中文回答。
- 14 个业务实现占位文件保持零字节；未启动 Go、数据库、MinIO 或前端，也未验证尚未实现的业务工具。
