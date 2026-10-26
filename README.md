# 小猪小狗：独立业务后端与 pi 插件式接入

这是从原项目提取的新目录，可以整体复制出去独立开发。原项目未修改。这里包含 Go 业务后端、部署配置、商品原始资料和评测输入；不包含 Web 和 Android 客户端。

业务 API 可独立运行。项目已接入 pi 1.4.0，支持本地 CLI、SDK 单次对话以及 .pi 插件/技能加载；购物业务工具、记忆、RAG 和 HTTP/SSE 仍保留占位。旧聊天、Prompt 发布、向量检索和评测执行接口仍返回 `501 agent_not_configured`。

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
.pi/extensions/              pi 插件；状态工具已实现，业务/harness 保留占位
.pi/skills/                  有效 SKILL.md，业务流程占位
src/pi-session.mjs           pi SDK 会话初始化
src/                         HTTP、Go API 客户端、RAG 空占位
scripts/pi.mjs               单次对话和离线检查
quality/data/                原始商品资料、图片、评测 JSONL
deployments/                 独立 Compose
docs/                        API 清单、Agent 接入边界、迁移说明
```

HTTP handler 调用各业务 Service；Service 只依赖本域的窄 Repository 接口；MySQL 实现集中装配。登录密码规则、商品默认值、购物车 SKU 校验和优惠试算已放入 Service。下单、库存预留、订单项和购物车清理保持单一数据库事务，事务内部状态检查与 SQL 一起保留。其余原有业务行为尽量保持，未改造成新的交易系统。

## pi 本地接入

要求 Node.js 22.19+，在本目录执行：

```bash
npm ci
npm run check
npm run pi
```

也可以执行 `npm run chat -- "调用 shop_status，说明当前接入状态"`。默认使用本机已有的 `deepseek/deepseek-flash` 模型配置，凭证仍由用户 pi 管理，不进入仓库。离线 check 不调用模型、不需要启动后端。插件加载成功不代表购物业务已接通，详情见 [Agent 边界](docs/agent-boundary.md)。

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
