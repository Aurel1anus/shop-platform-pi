# 小猪小狗：独立业务后端与 pi 插件式接入

这是从原项目提取的新目录，可以整体复制出去独立开发。原项目未修改。这里包含 Go 业务后端、部署配置、商品原始资料和评测输入；不包含 Web 和 Android 客户端。

业务 API 可独立运行。项目已接入 pi 1.4.0，支持本地 CLI、SDK 单次对话以及 .pi 插件/技能加载；list_products、get_product 和 list_product_skus 已通过 Python 接通关键词分页查询、商品详情和规格查询，其余业务工具、记忆、RAG 和 HTTP/SSE 仍保留占位。旧聊天、Prompt 发布、向量检索和评测执行接口仍返回 `501 agent_not_configured`。

## 目录与职责

```text
backend/
  cmd/api/                    启动入口
  cmd/import-products/        原始商品与知识资料导入
  internal/app/               依赖装配
  internal/httpapi/           按业务拆分的 HTTP handler、鉴权、SSE 预留边界
  internal/auth/              登录、注册、密码策略、账号操作
  internal/catalog/           商品、SKU、类目、商家
  internal/cart/              加购校验、购物车、优惠试算
  internal/order/             订单业务入口
  internal/promotion/         促销与优惠券
  internal/review/            评价
  internal/knowledge/         原始资料、清洗入库；不切块
  internal/files/             文件元数据
  internal/domain/            共享业务类型
  internal/platform/mysql/    按业务拆分的 SQL 与原子事务
  internal/platform/configcenter/ 内存配置与可选 Nacos
  internal/platform/objectstore/  MinIO
  migrations/                嵌入式建表 SQL 与独立演示数据
.pi/extensions/              状态、商品列表、详情与规格工具已实现，其余业务/harness 占位
.pi/skills/                  shopping 指导关键词分页、详情与规格查询；其余业务流程占位
src/pi-session.mjs           pi SDK 会话初始化
src/clients/commerce.py      Python 只读商品详情、列表与规格 HTTP 客户端
src/clients/commerce-python.ts  详情、列表与规格的 Python 子进程薄适配
src/                         HTTP、RAG 仍为空占位
scripts/pi.mjs               单次对话和离线检查
quality/data/                原始商品资料、图片、评测 JSONL
deployments/                 独立 Compose
docs/                        API 清单、Agent 接入边界、迁移说明
```

HTTP handler 调用各业务 Service；Service 只依赖本域的窄 Repository 接口；MySQL 实现集中装配。登录密码规则、商品默认值、购物车 SKU 校验和优惠试算已放入 Service。下单、库存预留、订单项和购物车清理保持单一数据库事务，事务内部状态检查与 SQL 一起保留。其余原有业务行为尽量保持，未改造成新的交易系统。

新增 Agent 侧业务接入、检索与数据处理逻辑优先用 Python，pi 会话与扩展注册继续使用 JavaScript/TypeScript。Python 通过 HTTP 调用 Go，不直接访问业务表。三个商品工具已接通 pi 到 Python 的子进程调用，没有 Python 常驻服务。

使用 Python 3.10+，在项目根目录查询商品（前提是 Go API 和数据库已启动）：

```bash
python3 -B src/clients/commerce.py p_002
```

默认使用 `COMMERCE_API_BASE_URL`，未配置时为 `http://127.0.0.1:8081/api/v1`；可通过 `--base-url` 覆盖。正常结果为商品 JSON；失败向 stderr 输出错误 JSON，退出码为 1。`--timeout` 默认 5 秒，表示 socket 等待超时。pi 适配层为整次子进程调用设置 8 秒超时，收到取消信号或超时时终止子进程。默认运行 python3，可用应用环境变量 `COMMERCE_PYTHON` 指定解释器路径，不通过模型参数设置。

Python 模块另提供 `list_products(keyword="", *, page=1, page_size=10, base_url=None, timeout=5)`，可通过 `from src.clients.commerce import list_products` 导入，返回含 items、total、page、page_size 的完整分页字典。空关键词用于浏览全部商品；页码范围为 1～100000，每页数量为 1～100，均不接受布尔值。无匹配返回空 items，输入错误、请求失败或返回数据格式异常抛出 CommerceError。列表已接入 Python CLI、TS 适配及 pi 工具。

TS 适配提供 `listProducts({ keyword, page, pageSize }, { signal, timeoutMs })`，两组参数均可省略。它与 getProduct 共用子进程执行、错误转换、整体超时和取消逻辑；返回保留后端的 items、total、page、page_size 字段。关键词使用独立参数 `--keyword=值` 传给 Python，避免以 -- 开头的内容被误当成选项，不经过 shell。商品参数范围与返回字段仍由 Python 校验。

在项目根目录、Go API 和数据库运行时，查询列表：

```bash
python3 -B src/clients/commerce.py --list --keyword "Camera & Mouse" --page 2 --page-size 1
```

CLI 的详情商品 ID、`--list`、`--skus PRODUCT_ID` 三选一；商品列表省略关键词可浏览全部商品，默认第一页、每页 10 条。`--keyword` 仅用于商品列表；`--page`、`--page-size` 可用于商品及规格列表。成功向 stdout 输出 JSON，退出码为 0（包括空页）；业务校验或请求失败向 stderr 输出错误 JSON，退出码为 1；缺失/冲突参数或无法解析的整数由 argparse 输出用法错误，退出码为 2。

在相同目录及服务前提下，查询规格：

```bash
python3 -B src/clients/commerce.py --skus p_002 --page 1 --page-size 10
```

Python 提供 `list_product_skus(product_id, *, page=1, page_size=10, base_url=None, timeout=5)`；TS 提供 `listProductSkus(productId, { page, pageSize }, { signal, timeoutMs })`。返回完整分页数据，每个 SKU 包含 skuId、productId、skuName、price、stockQuantity、stockStatus、specs，并校验商品归属。需要全部规格时按 total 继续翻页。当前 Go SKU 接口对不存在商品也返回空列表，部分数据库错误也可能被转换为空结果；不能据此判断商品不存在或宣称库存为零，需要确认商品时另查 get_product。此次保留后端行为，未修改 Go。

在根目录执行（需要 Node 依赖、Python 3.10+、Go 后端及已配置的模型凭证）：

```bash
npm run chat -- "调用 get_product 查询商品 ID p_002，报告实际名称和价格"
```

预期出现 `[tool] get_product`，随后模型根据真实结果回答。也可以查询列表：

```bash
npm run chat -- "只调用一次 list_products，关键词 Quiet，第一页、每页 1 条，报告实际名称、价格和总数"
```

预期出现 `[tool] list_products`。工具参数为 keyword、page、page_size，默认空关键词、第一页、每页 10 条；结果包含 items、total、page、page_size，文本与 details 均保留完整分页结果。多词查询可能匹配任意词，尚非语义推荐；写操作未接入。

模型规格查询可执行：

```bash
npm run chat -- "调用 list_product_skus 查询 p_002，报告实际规格、价格、库存数量和规格总数"
```

预期出现 `[tool] list_product_skus`。参数为必填 product_id 和可选 page、page_size；文本及 details 保留完整分页结果。不要将商品列表中的单个 skuId 当作全部可售规格。

## pi 本地接入

要求 Node.js 22.19+，在本目录执行：

```bash
npm ci
npm run check
npm run pi
```

也可以执行 `npm run chat -- "调用 shop_status，说明当前接入状态"`。默认使用本机已有的 `deepseek/deepseek-flash` 模型配置，凭证仍由用户 pi 管理，不进入仓库。离线 check 不调用模型、不需要启动后端。插件加载成功不代表购物业务已接通，详情见 [Agent 边界](docs/agent-boundary.md)。

SDK 入口（chat/check）只加载明确指定的本项目扩展和 .pi/skills，关闭其余扩展、技能、上下文文件及提示词模板的自动加载；模型凭证来源保持不变。npm run pi 的原生 CLI 入口仍保留默认发现规则。npm run check 同时校验实际扩展/技能路径和空上下文列表。

## 本地启动

要求 Go 1.24+ 和 Docker Compose。新服务默认端口为 `8081`，数据库为 `xzxg_commerce`，宿主机 MySQL 端口为 `3307`，避免与原项目默认环境冲突。

从本目录执行：

```bash
docker compose -f deployments/docker-compose.yml up -d
cp .env.example .env
set -a
. ./.env
set +a
cd backend
go run ./cmd/api
```

程序读取环境变量，不会自动加载 `.env`。开发环境默认自动建表并写入演示数据；生产环境默认关闭两者。迁移面向全新数据库，不包含旧项目数据库升级脚本，勿指向旧数据库。

```bash
curl http://localhost:8081/api/v1/health
```

开发演示账号沿用 `admin / admin123456`、`merchant / merchant123456`、`user / user123456`。

商品原始语料已复制到本目录。进入 `backend` 后导入：

```bash
go run ./cmd/import-products
```

导入器保存商品、SKU 以及商品说明、FAQ、评价原文。知识资料状态为 `pending_index`，`chunk_count` 为 0，metadata 包含所属 `product_id`；不计算 embedding、不建立索引。商品 JSON 内的评价被作为知识原文导入，不会冒充已购买订单生成交易评价。

## Docker 运行 API

在本目录执行：

```bash
docker compose -f deployments/docker-compose.yml --profile app up -d --build
```

API 镜像包含商品图片与原始 JSON，数据库迁移 SQL 嵌入二进制。容器首次启动只写入少量演示商品；导入完整商品集可在本地运行上述导入命令，连接 `3307` 数据库。

文件上传需要 MinIO，控制台为 `http://localhost:9101`。语音代理保留，使用时另配语音供应商凭证。

MinIO 使用本地构建的镜像：从官方 GitHub 下载固定的 `RELEASE.2025-04-22T22-12-26Z` 源码，校验 SHA-256 后编译。这样不再依赖已无法匿名拉取的 MinIO 发行镜像；首次启动会下载 Go 构建依赖并花费更多时间，后续复用 Docker 构建缓存。

Nacos 可选：

```bash
docker compose -f deployments/docker-compose.yml --profile config up -d
```

本地 Go 进程设置 `NACOS_ADDR=http://127.0.0.1:18848` 后重启。独立配置组为 `XZXG_COMMERCE`。未配置 Nacos 时，管理员配置修改仅存于内存，重启恢复环境变量与默认值。

## 验证

在项目根目录执行商品客户端离线测试（Python 3.10+，仅标准库，不需要启动后端）：

```bash
python3 -B -m unittest discover -s tests -p 'test_*.py'
```

此测试模拟 HTTP 响应，覆盖详情、列表及规格查询、输入校验、HTTP 错误、连接失败、无效数据、超时与命令行输出；`npm run check` 仍只检查 pi 接入，两者需分别执行。

完整商品工具离线回归在根目录执行 `npm run check:commerce`，要求 Node 依赖、Python 3.10+ 及本机临时端口监听权限。它先执行 Python 单元测试，再通过真实 pi 详情/列表/规格工具、TS 适配和 Python 子进程访问本机模拟 HTTP 服务，覆盖正常结果、失败、整体超时、取消和解释器缺失；列表及规格另覆盖分页、默认参数、空页，规格校验商品归属与库存字段。不调用模型、不要求 Go 或数据库。预期全部通过，不能据此推断真实模型端到端通过。

固定只读验收在项目根目录运行 `npm run check:commerce:live`。需要 Go API、MySQL 和包含 p_002 及至少两个商品的开发演示数据；先运行上述离线回归，再执行真实工具→Python→Go 的五个场景（查询/详情/规格、翻页、无匹配、缺失商品、SKU 空页）。全部通过时退出码为 0；不会自动启动服务或跳过失败，不请求模型。场景来源、执行前提及首次基线见 [quality/README.md](quality/README.md)。

在 `backend` 执行：

```bash
go test ./...
go vet ./...
go build ./...
```

普通测试不需要中间件，覆盖架构依赖、登录密码迁移、购物车 SKU 校验、优惠试算、HTTP 权限、语音协议和文本清洗。真实数据库事务测试需要一个专用的空测试数据库；明确设置 `COMMERCE_TEST_MYSQL_DSN` 才会执行，否则跳过。

```bash
COMMERCE_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3307)/xzxg_commerce_test?parseTime=true&loc=Local' go test ./internal/platform/mysql -run TestCheckoutIntegration -v
```

测试数据库须预先创建。测试会在该库建表并插入演示数据，请勿使用生产库。

## 业务边界

- 支付仍为虚拟支付；售后退款和第三方支付未实现。
- 优惠试算保留，实际下单仍沿用原代码的零折扣行为，未实现优惠核销闭环。
- 金额计算、订单取消/超时后的库存补偿等沿用原实现，需要在真实交易前另行完善。
- 存储接口的大部分错误仍沿用 `bool` 返回；本次没有批量改动 API 错误语义。
- 商品列表只有 SQL 关键词与类目筛选，没有旧 Agent 的重排和语义相关性判断。
- 旧 Agent 的输入风控、商品引用白名单和工具调用预算需要在新 harness 实现；业务 API 的角色和资源归属校验保留。
- 生产模式需要独立数据库、明确 CORS 来源、非默认 MinIO 凭证，且禁止写入演示数据。不要求模型或 Milvus 密钥。

pi 接入及后续业务边界见 [Agent 边界](docs/agent-boundary.md)，业务接口见 [API 清单](docs/api.md)。

后续开发顺序、各阶段交付和验收标准见 [开发蓝图](docs/development-blueprint.md)。
