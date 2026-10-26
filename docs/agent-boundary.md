# Agent 接入边界

`agent/src` 的 `.ts` 与 Prompt 文件为零字节占位，当前没有 TypeScript 运行工程或 pi 依赖。后续应锁定选用的 pi 版本，再建立 package.json 和运行配置。

## 运行与数据归属

- Go：账号、商品、库存、购物车、订单、优惠券、评价、文件与原始知识资料。
- harness：会话、完整模型对话、工具调用及结果、run、trace、用户授权上下文、预算和取消。
- RAG：原文清洗后的切块、embedding、检索索引、重排与评测适配。

Agent 使用 `clients/commerce.ts` 调用 Go 业务 HTTP API，不直接写业务表。用户身份来自已认证的请求上下文；不要把任意 account_id 作为模型可填写的工具参数，也不要用管理员凭证统一代办用户交易。

## 预留文件

| 文件 | 后续职责 |
|---|---|
| `server.ts` | Agent 内部请求入口 |
| `harness/run.ts` | 请求去重、会话串行、创建 run、停止和收尾 |
| `harness/context.ts` | 历史对话、需求摘要、最近推荐商品顺序 |
| `harness/policy.ts` | 工具白名单、业务授权上下文、调用预算 |
| `harness/events.ts` | pi 事件转换、客户端卡片、trace |
| `harness/store.ts` | 会话、消息、工具结果和 run 持久化 |
| `tools/catalog.ts` | 商品查询工具 |
| `tools/cart.ts` | 购物车与优惠试算工具 |
| `tools/order.ts` | 下单与订单操作工具 |
| `tools/knowledge.ts` | 自写 RAG 查询工具 |
| `clients/commerce.ts` | Go API 客户端 |
| `retrieval/ingest.ts` | 原文入索引 |
| `retrieval/search.ts` | 召回与重排 |
| `retrieval/types.ts` | 文档、片段和来源结构 |
| `prompts/system.md` | 系统提示词 |

## 客户端兼容

旧聊天入口、评测和 Prompt/trace 页面相关 API 当前明确返回 501。没有空实现返回伪成功，也没有自动转发到不存在的 Agent 服务。

后续可由 Go 在统一鉴权后转发聊天请求，或让网关按路径分流到 Agent；两种方式应只选择一个会话/run 数据拥有者。若复用旧 Android，需要适配原 `message_start`、`text_delta`、`block_delta`、`followups`、`message_end` 等 SSE 事件。当前已移除旧自定义 `<final>` / `<item>` 解析器。

商品和订单卡片应来自工具实际结果。旧 Agent 中的相关商品白名单、指代解析、明确加购/结算意图检查不会由普通业务 HTTP API 自动提供，需要新 harness 实现。

写工具的重试需要稳定的业务幂等键；现有 HTTP 写接口未新增通用幂等协议。取消生成只能停止后续执行，不能撤销已经提交的交易。

## 知识与评测

管理端/商家端的 documents 列表返回原始 `content`、metadata、来源和 hash。自写 RAG 可以分页读取，但这些管理接口需要对应角色凭证，不能将管理凭证暴露给模型。

新数据库不创建旧 Agent 表和 knowledge_chunks。商品导入器在知识 metadata 中记录 product_id，原始语料同时保留在 quality/data 中。

评测 JSONL 保留原数据。300 条检索用例的 chunk_id 属于旧切块方式，重切后应按 product/source ID 评分。Agent 场景应检查业务结果；旧 memory trace 字段不要当作新实现的必需协议。
