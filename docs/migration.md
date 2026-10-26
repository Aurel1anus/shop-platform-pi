# 从原项目迁移

此目录是独立的新工程，原 backend、frontend、android-native 与 deployments 未修改。

| 原位置 | 新位置/处理 |
|---|---|
| backend/cmd/api/main.go | backend/cmd/api + internal/app，移除模型和向量启动依赖 |
| backend/src/httpapi/server.go | internal/httpapi，按账号、商品、购物车、订单、优惠、评价、知识、文件、配置拆文件 |
| backend/src/store/interface.go | 拆为各业务 Service 自己声明的窄 Repository 接口 |
| backend/src/store/mysql.go | internal/platform/mysql，按业务拆 SQL，保留下单等原子事务 |
| 登录与密码 helper | internal/auth/service.go |
| 商品默认值 | internal/catalog/service.go |
| 优惠试算 | internal/cart/service.go |
| backend/src/domain | internal/domain，只保留业务类型 |
| backend/src/ingest | internal/knowledge/ingest，保留清洗、URL 获取与来源约束 |
| backend/src/configcenter | internal/platform/configcenter，仅业务/基础设施默认配置，Nacos 配置组独立 |
| backend/src/objectstore | internal/platform/objectstore |
| backend/src/agent | 不复制，agent/src 留空等待新实现 |
| backend/src/rag、imagevector、retrievalconfig | 不复制，retrieval 预留空文件 |
| Agent 会话、run、trace、Prompt 表 | 新数据库不创建，由新 harness 自行设计 |
| knowledge_chunks | 新数据库不创建，原文存入 knowledge_documents |
| backend/migrations | 全新业务库建表与演示数据分离，SQL 嵌入二进制 |
| quality/data | 原样复制，图片与 JSONL 保留 |

新目录默认 API 8081、MySQL 3307、MinIO 9100/9101，数据库 xzxg_commerce。要让旧客户端连接，先修改 API 地址；聊天与 AI 管理页面需要等新 Agent 接入。

这次保留了原业务的虚拟支付、优惠核销未接通、bool 错误返回等边界，详情见根 README。仓库层仍保留与原子 SQL/事务紧密相关的状态检查和部分数据装配，未为每个读接口另造一套 DTO 或持久化模型。
