# 学习进度

## 当前阶段

蓝图第一阶段收尾：三个只读商品工具及固定工具/服务验收已完成，真实模型“列表→详情→规格”链路通过。仍待开发者独立验收练习；批量模型场景评分尚未实现，不自动进入网页或写工具阶段。

## 已验收内容

- 2026-10-26：Node.js v22.23.4 下执行 `npm run check` 通过。
- 本次检查实际初始化 pi SDK，加载 3 个扩展和 3 个技能，执行 shop_status。
- 检查未调用模型、未探测 Go 后端；不能据此认定商品查询或数据库连接可用。
- 2026-10-26：执行 `docker compose -f deployments/docker-compose.yml up -d --wait mysql`，本项目 MySQL 容器健康检查通过。
- 在 backend 使用已有 Go 1.24 工具链启动 `go run ./cmd/api`，实际收到监听 :8081 的日志；启动流程连接数据库，并按开发默认配置执行建表和演示数据初始化。
- 真实 HTTP 联调：`curl -i http://127.0.0.1:8081/api/v1/health` 返回 HTTP 200 和 `{"status":"ok"}`。健康处理函数只返回固定状态；本次未验证商品接口、模型端到端、MinIO，也未运行 Go 全量测试（本次无 Go 代码修改）。
- 本次结束时保留 Go API 进程和 MySQL 容器运行，便于开发者手动练习。终端 PATH 尚无 go，当前使用 /private/tmp/xzxg-go-toolchain/go/bin/go，GOPATH 与 GOCACHE 分别位于 /private/tmp/xzxg-go-path 和 /private/tmp/xzxg-go-cache。
- 商品列表真实联调：GET /api/v1/products?page=1&page_size=2 返回 HTTP 200，items 含 Y Camera Max 与 Quiet Mouse S；keyword=Quiet 筛选返回 Quiet Mouse S，价格为 129.00。两次响应的 total 均为 0，与非空 items 不一致，分页总数存在问题，未修复。
- 后续已修复上述 total 问题：提取 productCardFrom 复用关联表与筛选条件；计数失败按现有接口约定返回 nil、0，不继续返回非空列表。
- 修复验证：backend 的 go test ./...、go vet ./...、go build ./... 均通过；普通测试未设置数据库变量，数据库集成用例跳过。另在独立 xzxg_commerce_test 库运行 TestProductPaginationIntegration，关键词、类目、无匹配、不同页大小及越界页检查通过。
- 重启 API 后真实 HTTP 验证：每页两条商品时 total=2；keyword=Quiet 时返回一件商品、total=1。API 继续运行，未执行模型端到端验证。

## 练习情况

- 已提供入口追踪练习：从 package.json 的 check 命令找到 scripts/pi.mjs 的离线检查分支。
- 开发者已提交入口追踪结果，尚未独立修改代码。
- 开发者已正确判断 chat 模式下 checking 为 false，并找到创建会话的调用；已确认理解交互 CLI 与脚本接入的用途区别，尚未独立修改代码。
- 当前阅读 src/pi-session.mjs：配置读取、项目资源加载与内存会话创建。
- 开发者已找到默认模型配置，能判断仅添加工具名称而未实现注册不能查询商品。
- 健康接口由助手完成真实联调；开发者尚未提交独立请求和路由追踪结果。
- 开发者已正确解释端口用于找到后端服务、路径用于匹配处理函数；补充了路由同时匹配 HTTP 方法。当前练习是独立修改商品列表的 keyword 查询参数。

## 待解决问题与下一步

- 确认开发者能解释离线检查的验证范围，以及 pi.mjs 与 pi-session.mjs 的职责区别。
- 当前练习：手动请求健康接口，并在 backend/internal/httpapi/server.go 找到路由注册。
- 商品列表查询与 keyword 筛选已由助手完成真实验证；待开发者提交独立练习结果。
- 开发者独立查询 keyword=Camera，报告商品名称 Y Camera Max，查询参数练习完成。
- total 原因：ListProductsPage 替换 SQL 时删掉 FROM/JOIN，且忽略 Scan 错误。开发者正确解释了修改 total 初始值不能修复查询失败；修复及回归已由助手完成，不等于开发者独立掌握修复。
- 已知限制：当前 Repository 返回值没有 error，数据库失败仍不能通过 HTTP 与空结果区分；本次未扩大为全项目错误接口重构。
- 商品详情与 SKU 已由助手真实 HTTP 验证：/products/p_002 返回 Y Camera Max、价格 3499.00；/products/p_002/skus 返回一个标准版 SKU（sku_002），规格库存 32，两次均为 HTTP 200。开发者尚未提交独立规格查询练习。
- 按开发者要求，新增 Agent 侧具体业务逻辑优先使用 Python，Go 业务后端保留，pi 会话与扩展注册继续用 JS/TS；已写入 AGENTS.md 并同步 README、蓝图和接入边界。
- commerce 已迁移为 src/clients/commerce.py 的 get_product，原 TypeScript 实现和 Node 测试移除；仅标准库，支持公开详情查询、输入校验、默认 5 秒 socket 等待超时、错误转换及 CLI JSON 输出。正常输出到 stdout，失败错误输出到 stderr，退出码为 1。
- Python 3.12.3：`python3 -B -m unittest discover -s tests -p 'test_*.py' -v` 的 7 项离线测试通过；`npm run check` 通过。真实 Python/Go 联调查询 p_002 返回 Y Camera Max、3499.00；不存在商品返回“商品不存在”且退出码为 1。未做模型端到端验证，本轮未改 Go，未重跑 Go 检查。
- 当前学习：Python get_product 如何经 HTTP 调用 Go；错误通过 CommerceError 抛出，CLI 转为 JSON 错误和失败退出码。实现由助手完成，尚未确认开发者独立掌握。
- 已实现 commerce-python.ts 子进程适配：固定 Python 脚本、参数数组、默认 8 秒整体超时、取消时终止进程；COMMERCE_PYTHON 由应用配置解释器路径。Python 仍负责 HTTP 和业务数据校验，未增加依赖或常驻服务。
- get_product 已在 commerce/catalog.ts 注册并加入默认名单，返回文本和 details.product；shop_status、SYSTEM 与 shopping 技能同步实际能力，仅支持已知 ID 详情查询。
- 本轮验证：npm run check 通过；npm run check:commerce 通过（7 个 Python 单元测试，以及真实 pi/Python 到本机模拟 HTTP 的 6 个子用例：正常、关键失败、整体超时、取消、解释器缺失、连接失败）。模型不参与这些离线测试。
- 真实模型端到端通过：执行 npm run chat 查询 p_002，观察到 [tool] get_product，最终回答 Y Camera Max、3499.00 元。该次实际访问 Go 后端；未做商品搜索、完整 SKU 或写操作验证。本轮未修改 Go，未重跑 Go 检查。
- 开发者已正确解释 Python 仅向后端请求，补充 Go 查询的是 MySQL。当前理解检查：工具失败时为何不能让模型用猜测价格替代。实现和测试由助手完成，尚未确认开发者独立掌握。
- 下一小目标：在 Python 中实现商品列表查询，再以薄适配注册 list_products；不提前开放写工具。
- 按开发者要求，适配文件从 .mjs 改为 commerce-python.ts，补充参数和结果类型，更新扩展引用及 Node 测试加载参数；npm run check、npm run check:commerce 均通过。未执行独立 TS 静态类型检查或重新调用真实模型，尚未迁移 Python RPC 入口。
- 当前讨论：新工具仍需 Python 业务实现和 TS 工具注册，子进程执行机制可共享；目前仅 getProduct，新增第二个工具时再提取共用调用函数，避免每个工具复制进程控制逻辑。
- 2026-10-27：SDK 资源加载改为显式名单：noExtensions/noSkills/noPromptTemplates 关闭自动发现，明确指定三个项目扩展及 .pi/skills，继续禁用上下文文件和主题自动加载。模型凭证来源未改，原生 npm run pi 的发现规则未改。
- 验证：npm run check 校验实际扩展、技能文件路径及空上下文/模板列表，通过；临时向 SDK agentDir 注入额外扩展和技能后仍通过，临时文件已清理。npm run check:commerce 全部通过。本次未请求模型或真实 Go API，未修改 Go 代码。
- 当前理解检查：noSkills 是关闭自动发现，不阻止 additionalSkillPaths 显式加载；资源名单不限制 read 工具的文件访问权限。
- 2026-10-27：开发者独立将扩展名单改为目录路径。实际运行 npm run check 发现 SDK 将裸 .pi/extensions 当作模块加载并失败；此前助手关于该 SDK 入口直接支持扩展目录扫描的建议不准确。
- 已由助手修正为 additionalExtensionPaths 指向 .pi 资源包，新增 .pi/package.json，通过 pi.extensions: ["extensions"] 声明扩展目录；保留技能目录配置及检查脚本的具体路径名单。同步更新 agent-boundary.md。
- 修正后 npm run check 通过（3 个扩展、3 个技能）；npm run check:commerce 通过（7 个 Python 单元测试及 6 个工具联调子用例）。未请求模型或真实 Go API，未修改 Go。目录配置由开发者尝试，资源包修正由助手完成，尚未确认独立掌握。
- 当前理解检查：.pi/package.json 中 extensions 路径相对哪个目录；扫描目录与检查固定名单各自负责什么。下一步仍为 Python 商品列表查询，尚未开始实现。
- 当前练习推进到商品列表 URL 构造：先在 Python 交互环境用 urlencode 编码 keyword、page、page_size，再拼接 /products 请求地址；以 Camera 和包含空格、& 的关键词验收。助手已核对标准库编码输出，开发者尚未提交练习；未修改商品客户端或注册 list_products。
- 开发者已提交 urlencode 与 URL 拼接代码，含空格和 & 的关键词编码正确；可直接 urlencode(params)，无需重建相同字典。已提交 Python 请求后读取 total/items 的结果：total=2，含 Y Camera Max、Quiet Mouse S。请求读取和 JSON 取值练习完成；尚未提交实际请求 URL，不能据此确认 Camera 单关键词筛选。后端会将多词关键词拆分后按 OR 匹配，Camera & Mouse 可能匹配两件商品；下一步打印 request.full_url 并对照单关键词结果，暂未新增正式函数。
- 开发者确认实际关键词为 Camera & Mouse，与后端分词 OR 匹配行为一致；已提交正确的 list_products 练习函数，尚未提交该函数的分页运行结果。后续教学偏好：助手直接实现完整小目标，再提理解问题，减少逐行填空。
- 按开发者要求在 commerce.py 完成正式 list_products：校验关键词/分页参数，复用 _request_json 的地址、超时、HTTP 与 JSON 错误处理，校验分页元数据及商品 ID/名称/价格字段。空列表是正常结果，列表 HTTP 404 为请求失败，详情 404 仍为商品不存在。未修改详情 CLI 协议或注册新 pi 工具。
- 验证：npm run check:commerce 通过（11 个 Python 测试及现有详情工具 6 个联调子用例）。获准访问本机后真实 Go 联调通过：Camera & Mouse 每页一条，第一页 Y Camera Max、第二页 Quiet Mouse S，两页 total=2；随机无匹配关键词返回 total=0、items=[]。未请求模型，未修改 Go。本次实现由助手完成，理解待确认。
- 当前理解问题：为何分页排除 bool；为何空列表不抛异常但 HTTP 500 抛异常。下一步接通列表 CLI 与 TS 薄适配，再注册 list_products，当前尚未开始。
- 开发者希望理解题避免重复已解释内容；对“HTTP 200 但 price 为数字”的追踪题暂未答出，助手已解释 HTTP、JSON 解析和字段校验的区别，尚未确认独立掌握。
- 本轮完成列表 CLI：--list 与详情 ID 互斥，支持 --keyword/--page/--page-size；保留详情原调用方式及 -- 分隔 ID 的行为。成功 JSON 到 stdout/退出 0，CommerceError JSON 到 stderr/退出 1，argparse 用法错误到 stderr/退出 2。实现由助手完成，未新增依赖或修改 TS/Go。
- 本轮 npm run check:commerce 通过（13 个 Python 测试、现有详情工具 6 个联调子用例）。真实 CLI 查询第二页返回 Quiet Mouse S、total=2；无匹配返回空列表且退出 0；分页越界退出 1；非法整数退出 2；不可用地址请求失败退出 1（本机代理返回 HTTP 502，未据此认定为直连拒绝）。未请求模型。
- 当前理解题：为什么仅在 Python CLI 注册列表仍不能让模型使用 list_products。推荐下一步：共享 TS 子进程调用机制并增加列表薄适配，再注册 pi 工具，尚未开始。
- 开发者已正确解释模型需要工具名称、描述和注册，跨语言调用需 TS/Python 薄适配；补充参数定义和执行函数也属于注册内容。
- 本轮在 commerce-python.ts 增加 listProducts({ keyword, page, pageSize }, { signal, timeoutMs })，提取私有 runPython<T> 共用执行、超时、取消与错误处理；关键词使用 --keyword=值，保留详情 -- ID 调用。未新增依赖、未修改 Python/Go，未注册列表工具。
- 验证：npm run check 通过；npm run check:commerce 通过（13 个 Python 测试及 8 个 Node 联调子用例），新增列表经真实 Python 到模拟 HTTP 的关键词/默认参数/分页/空结果/错误检查，共享超时、取消、解释器缺失和断连用例覆盖详情及列表。真实 TS→Python→Go 联调返回第一页 Y Camera Max、第二页 Quiet Mouse S、两页 total=2，无匹配 total=0；非法 pageSize=0 明确失败。未执行独立 TS 静态类型检查，未请求模型。
- 当前理解题：把 --keyword=值 改为 --keyword、值 两个参数时，关键词 --help 会怎样。推荐下一步：注册 pi list_products 工具并更新技能、状态及离线检查，尚未开始。
- 开发者暂未答出 --keyword 与 --help 的参数解析问题；助手已解释等号将选项和值绑定，尚未确认独立掌握。
- 本轮完成 pi list_products 注册：参数 keyword/page/page_size 可省略，分页 schema 约束范围；执行时映射 page_size 到 pageSize、传递取消信号，文本及 details 返回完整分页结果。默认工具、shop_status、SYSTEM、shopping 和离线检查同步更新；未新增依赖、未修改 Python/Go 或开放写工具。
- 本轮 npm run check 通过；npm run check:commerce 通过（13 个 Python 测试及 9 个 Node 联调子用例），新增真实列表工具参数映射、默认参数及空结果检查，失败、取消、解释器缺失和断连经过真实列表工具验证。
- 真实模型端到端通过：npm run chat 指定 Quiet、第一页每页一条，出现 [tool] list_products，最终回答 p_mouse_001、Quiet Mouse S、129.00、total=1、本页数量=1。未验证模型连续翻页或列表到详情多工具链路，未做独立 TS 静态检查。本次实现和验证由助手完成，不等于开发者已独立掌握。
- 当前理解问题：模型只传 keyword 时，默认 page/page_size 最先在哪层补齐。推荐下一步：实现只读 list_product_skus，继续第一阶段，尚未开始。
- 开发者正确回答列表默认 page=1/page_size=10 最先在 TS 薄适配层补齐，已能追踪这一处参数变化；询问商品列表与 SKU 区别，助手已结合现有默认 skuId 与多规格示例解释。
- 按开发者要求完成完整 SKU 接入：Python list_product_skus、CLI --skus PRODUCT_ID、TS listProductSkus、pi list_product_skus 及默认工具名单；复用 HTTP/子进程机制和分页校验。校验 SKU 归属、价格字符串、非负整数库存及 specs 字典，文本和 details 返回完整分页结果；状态、SYSTEM、shopping 及文档同步更新。未新增依赖、未改 Go，未开放写工具，保留开发者 test_learn.py。
- 本轮 npm run check 通过；npm run check:commerce 通过（18 个 Python 测试、11 个 Node 联调子用例），覆盖多 SKU 分页、默认参数、空/越界页、特殊 ID、错误归属、零库存、异常响应、请求失败、超时、取消、解释器缺失及断连。
- 真实 Go 联调：Python CLI 与 TS 查询 p_002，返回 sku_002、Y Camera Max 标准版、3499.00、库存 32；越界页为空但 total=1，不存在商品为空且 total=0；非法分页明确失败。现有 Go SKU handler 对不存在/无规格/部分存储错误均可能返回空结果，已明确保留并记录，不能把空结果当成商品不存在或库存零。
- 真实模型端到端通过：依次出现 [tool] list_products、[tool] get_product、[tool] list_product_skus，最终回答实际商品及标准版 SKU、价格 3499.00、库存 32，规格总数/本页数量均为 1。未验证真实多 SKU 商品模型连续翻页，未做独立 TS 静态检查；Go 未修改，未重跑 Go 检查。
- 本轮实现和验证由助手完成，尚未确认开发者独立掌握 SKU 接入。当前理解问题：为什么必须校验每个 SKU 的 productId 等于请求的商品 ID。推荐下一步：固定只读冒烟场景和验收基线，先完成第一阶段收尾，不自动进入第二阶段。
- 开发者回答 SKU 归属检查为“不是一个版本的商品”；助手纠正为属于另一件商品，区分 productId 与 skuId，尚未独立确认掌握。
- 本轮新增 quality/readonly-smoke.test.mjs 和 npm run check:commerce:live，沿用 node:test 与已注册工具，不新增依赖/评测框架或模型循环。改编现有查询/详情、无候选场景意图，补充分页、缺失商品与 SKU 空页，共五个真实服务场景；前置复用已有离线失败测试，未停止服务制造故障。
- 实际执行 npm run check:commerce:live：18 个 Python 测试、11 个模拟 HTTP 子用例、5 个真实工具→TS→Python→Go 场景全通过。快照 p_002 / Y Camera Max / 3499.00，sku_002 标准版价格 3499.00、库存 32。首次基线、场景来源和前提记录在 quality/README.md；未调用模型，不等于固定场景已通过批量模型验收。
- 本轮仅新增验收入口与文档，未修改业务逻辑或 Go；实现/执行由助手完成，开发者尚未独立运行。当前理解题：live 验收通过能否证明模型会选择正确工具，为什么。推荐下一步：开发者独立运行验收并用自然语言查询，核对工具调用与最终回答，再决定后续阶段。
