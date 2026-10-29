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
        index.ts                     注册三个只读商品工具
        catalog.ts                   list_products/get_product/list_product_skus，调用 Python 薄适配
        cart.ts                      空占位
        order.ts                     空占位
        knowledge.ts                 空占位
      harness/
        index.ts                     可加载的插件入口，事件钩子待实现
        context.ts / policy.ts       空占位
        events.ts / run.ts / store.ts 空占位
    skills/
      shopping/SKILL.md               关键词分页、详情与规格查询的流程与能力边界
      orders/SKILL.md                 购物车/订单技能占位说明
      knowledge/SKILL.md              知识检索技能占位说明
    runtime/                         本地运行缓存，Git 忽略
  src/
    pi-session.mjs                   已实现：创建 SDK 会话并加载本项目资源
    server.ts                        已实现：Node 内部固定 SSE 事件探针，尚未连接 pi
    clients/commerce.py               三个只读商品查询函数与 CLI
    clients/commerce-python.ts        子进程适配、整体超时和取消
    retrieval/                       空占位：ingest.ts、search.ts、types.ts
  scripts/pi.mjs                     已实现：单次对话与离线接入检查
```

pi 的扩展文件必须导出合法入口，技能文件必须有有效的 name 和 description。commerce 已注册 list_products、get_product 和 list_product_skus；harness 仍是可加载的无操作入口。其余空业务文件和技能占位说明不代表对应能力已经实现。

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

SDK 入口关闭扩展、技能及提示词模板的自动发现（noExtensions、noSkills、noPromptTemplates），通过 additionalExtensionPaths 指定本项目 .pi 资源包，由 .pi/package.json 的 pi.extensions 声明扫描 extensions 目录；通过 additionalSkillPaths 指定本项目 .pi/skills。扩展目录支持顶层 .ts/.js 文件及子目录中的 index.ts/index.js 等合法入口，不递归加载所有业务文件。新增入口无需修改 pi-session.mjs，但需更新 scripts/pi.mjs 的预期名单。当前 SDK 不支持直接把裸 .pi/extensions 目录传给 additionalExtensionPaths。这样不加载个人扩展或技能，也不自动发现 ~/.agents/skills 及祖先技能目录。noContextFiles 禁止自动加载 AGENTS.md/CLAUDE.md，noThemes 禁止自动加载主题。

SDK 仍通过独立的 ModelRuntime 读取用户模型凭证，资源隔离不改变凭证来源；使用进程内会话，不保存业务聊天历史。此配置仅作用于 npm run chat/check，npm run pi 仍采用原生 CLI 的资源发现规则。资源加载名单不是文件访问沙箱，read 工具仍有本地文件读取能力。它显式信任本仓库指定资源，这项配置针对本地开发入口，不是面向任意目录的信任开关。

默认模型为 `deepseek/deepseek-flash`，对应本机已有的 pi 设置。凭证通过 pi 用户配置或供应商环境变量提供，不复制进项目。没有凭证时先在 pi CLI 中 `/login` 配置；不要把密钥写进 SYSTEM.md 或 settings.json。

SDK 单次调用可临时选择模型：

```bash
PI_PROVIDER=deepseek PI_MODEL=deepseek-flash npm run chat -- "你好"
```

交互 CLI 用 pi 原生 `--provider` / `--model` 参数。SDK 对话超时为 60 秒，SDK 会话结束后释放；`check` 不请求模型 API，不要求后端或模型凭证。

默认可用工具为 `read`、`shop_status`、`get_product`、`list_products` 和 `list_product_skus`，没有启用 bash、edit、write。read 用于读取技能资料。当前入口是本地开发工具，尚不是具备用户鉴权和文件访问隔离的公网聊天服务。

## 实现与占位边界

| 部分 | 状态 |
|---|---|
| 项目本地 pi 安装、CLI、SDK 会话 | 已配置 |
| .pi 扩展和技能加载、系统提示词 | 已配置 |
| shop_status 工具 | 已实现，仅报告接入状态，不探测后端 |
| get_product | 已实现，按已知 ID 查询详情，返回文本与结构化商品 |
| list_products | 已实现，按关键词分页查询，文本与 details 均返回完整分页结果 |
| list_product_skus | 已实现，按商品 ID 分页查询 SKU 价格、规格及库存，校验商品归属 |
| 购物车、订单工具 | 占位 |
| 记忆、权限策略、业务会话/run/trace | 占位 |
| 自写 RAG | 占位 |
| HTTP/SSE | Go 鉴权转发与 Node 固定事件探针已实现；尚未连接 pi、网页与会话 |

`COMMERCE_API_BASE_URL` 默认 `http://127.0.0.1:8081/api/v1`，用于 shop_status 展示及 Python 商品客户端请求。客户端支持商品详情、关键词分页列表与 SKU 分页列表，默认 5 秒 socket 等待超时和错误转换；仅调用公开商品接口，不携带身份凭证。shop_status 报告 product_read_only 和三个商品工具名单，仍不探测后端，不能作为请求成功依据。

Python 3.10+ 客户端可通过 `python3 -B src/clients/commerce.py p_002` 执行，成功向 stdout 输出商品 JSON；失败向 stderr 输出错误 JSON 并以 1 退出。同步请求由 commerce-python.ts 适配调用，默认整次子进程调用上限 8 秒；超时或收到 AbortSignal 时通过 SIGKILL 终止子进程。解释器默认 python3，可由应用环境变量 COMMERCE_PYTHON 指定。调用使用固定脚本与参数数组，不经过 shell；命令行用 -- 分隔商品 ID 与选项。pi 原生加载 TypeScript 扩展，Node 测试入口通过 --experimental-strip-types 执行 TS，不增加编译依赖，也未新增静态类型检查步骤。

列表 CLI 使用 `python3 -B src/clients/commerce.py --list --keyword "Camera & Mouse" --page 2 --page-size 1`；与详情 ID 互斥。成功输出完整分页 JSON（含空结果），退出码为 0；CommerceError 输出 stderr 错误 JSON 并以 1 退出；命令行用法错误由 argparse 输出并以 2 退出。commerce-python.ts 提供 listProducts({ keyword, page, pageSize }, { signal, timeoutMs })，与 getProduct 共用文件内 runPython 子进程执行逻辑；列表用 --keyword=值 安全传递选项值，分页范围和业务字段由 Python 校验。pi list_products 工具参数为 keyword、page、page_size，均可省略；工具定义约束分页范围，执行时映射 page_size 到 TS 的 pageSize 并传递取消信号。结果 content 文本和 details 均含完整分页数据。商品名称查询先用列表获取真实 ID，不猜测 ID；当前不支持类目参数，关键词查询并非语义推荐。

规格 CLI 使用 `python3 -B src/clients/commerce.py --skus p_002 --page 1 --page-size 10`；与详情 ID、--list 互斥。Python list_product_skus 与 TS listProductSkus(productId, { page, pageSize }, { signal, timeoutMs }) 共用已有请求/子进程机制。pi 工具参数为 product_id 和可选 page、page_size，返回 content 文本及 details 的完整分页结果，每项含 SKU 标识、商品归属、名称、价格、库存数量/状态及 specs。全部规格需按 total 翻页，不能把一个默认 SKU 或第一页当成全部规格。

现有 Go SKU handler 不检查商品是否存在，空结果可能表示无规格、商品不存在、越界页或被旧存储接口吞掉的部分数据库错误；客户端保留接口语义，不将空结果转换为“商品不存在”或“库存为零”。确认商品存在需另调 get_product，越界页结合 total 判断。此次未改 Go 错误接口，相关局限仍保留。

## 数据和业务边界

- Go：账号、商品、库存、购物车、订单、优惠券、评价、文件和原始知识资料。
- Python：新增业务 HTTP 接入、检索和数据处理；不另建模型循环，不直接读写业务表。pi 会话及工具注册仍由 JS/TS 负责，三个只读商品工具已用子进程薄适配连接 Python，没有常驻服务。
- harness 扩展：后续实现会话上下文、商品指代、权限、预算、取消和业务持久化。
- RAG：后续实现原文切块、embedding、检索索引和重排，通过插件注册工具。
- skill：工作流指引；不替代业务工具，也不自行提供权限或交易保证。

未来工具通过薄适配调用 clients/commerce.py，再经 HTTP 访问 Go 业务 API，不直接写业务表。用户身份由已认证的请求上下文注入，不向模型开放任意 account_id，不共用管理员凭证代办交易。写工具仍需业务幂等，取消生成不会撤销已提交订单。

Go 的 `/api/v1/agent/*` 已改为鉴权后转发，使用 `AGENT_INTERNAL_URL` 配置内部地址；未配置返回 503，上游不可用返回 502。目前 Node 仅实现 `GET /api/v1/agent/events` 固定事件探针，其余路径返回 404，会话与聊天尚未接入。Node 默认只监听 `127.0.0.1:8082`（端口可用 `AGENT_PORT` 配置），不读取 pi/模型凭证。Go 覆盖内部身份头并移除登录凭证；该身份头只适用于可信内部部署。详情与验收流程见 README 和 api.md。Prompt、trace 和评测 API 继续保留 501。

## 知识与评测

Go 的 documents 管理接口保存和返回原始 content、metadata、来源与 hash；RAG 尚未切块或建索引。商品原文和评测输入在 quality/data 中。

旧 300 条检索用例的 chunk_id 对应旧切块，新实现应映射到来源或商品 ID。Agent 场景评测应检查真实业务结果，不能把占位插件成功加载当成业务通过。

## 验证记录

固定只读工具/服务验收入口为 npm run check:commerce:live，详见 quality/README.md。2026-10-27 实际执行：18 个 Python 测试、11 个模拟 HTTP 联调子用例及 5 个真实服务场景均通过。真实服务场景直接执行已注册的 pi 工具，不请求模型，不使用模拟数据冒充 Go 返回；不等于模型选择工具及最终回答已全部验收。

2026-10-27，规格工具接通后：npm run check 通过（5 个默认工具），npm run check:commerce 通过（18 个 Python 测试及 11 个 Node 联调子用例）。覆盖多个 SKU 分页、空页、默认参数、编码 ID、SKU 商品归属、零库存、异常字段、请求失败、超时、取消、解释器缺失和断连。真实 CLI/TS 查询 p_002 返回 sku_002、标准版、3499.00、库存 32；越界页为空且保留总数，不存在商品为空且总数为零。真实模型依次调用 list_products→get_product→list_product_skus，报告 Y Camera Max 标准版、价格 3499.00、库存 32，总规格数/本页数量均为 1。未验证真实多 SKU 商品的模型连续翻页；Go 未修改，未重跑 Go 检查，未做独立 TS 静态检查。

以下为列表接入时的历史记录：

2026-10-27，商品列表工具注册后验证：npm run check 通过（4 个默认工具）；npm run check:commerce 通过（13 个 Python 测试及 9 个 Node 联调子用例），覆盖真实列表工具参数映射、默认值、空结果、失败和取消。真实模型端到端出现 [tool] list_products，查询 Quiet、第一页每页一条，最终返回 p_mouse_001、Quiet Mouse S、129.00、total=1、本页数量=1。未验证模型连续翻页或列表到详情的多工具链路。

以下为商品详情接入时的历史记录，在 Python 3.12.3 / Node.js 22.23.4 验证：

- `npm run check` 通过：默认工具为 read、shop_status、get_product，不请求模型，也不启动 Python。
- `npm run check:commerce` 通过：7 个 Python 单元测试及 6 个工具联调子用例；真实 pi 注册和 Python 子进程访问本机模拟 HTTP，覆盖正常、失败、超时、取消及解释器缺失。无需模型或 Go。
- 真实模型端到端通过：SDK 对话观察到 `[tool] get_product`，工具经 Python 查询 Go API 的 p_002，最终回答 Y Camera Max、3499.00 元。商品搜索、完整 SKU 和写操作仍未接入。

以下为最初 pi 接入时的历史记录，不代表后续商品工具状态：

2026-10-26，在 Node.js 22.23.4 环境验证：

- `npm run check` 通过：真实 SDK 初始化、3 个扩展、3 个技能及状态工具执行；无模型 API 请求。
- `npm run pi -- --version` 返回项目本地版本 `1.4.0`。
- `npm run chat -- "只调用一次 shop_status 工具，然后用一句中文说明当前接入状态。"` 通过：使用已有 DeepSeek 凭证，模型实际调用 shop_status，并根据工具结果生成中文回答。
- 14 个业务实现占位文件保持零字节；未启动 Go、数据库、MinIO 或前端，也未验证尚未实现的业务工具。
