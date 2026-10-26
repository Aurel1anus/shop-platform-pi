# 原始语料和评测输入

本目录完整复制了原仓库 quality/data，未修改内容。

- ecommerce_agent_dataset：294 个商品 JSON、商品图片；JSON 保留完整商品介绍、SKU、FAQ 和评价，FAQ 共 827 对。
- eval：8 个 JSONL、共 477 条；其中 RAG 召回 300、意图 101、非导购意图 30、多轮记忆 10、端到端 8+20、无库存 5、图片搜索 3。

原始语料并非全部经过事实审核，部分为公开测试数据整理。FAQ 与商品说明可作为 RAG 原文，但不能把字段名 official_faq 当作官方认证。

旧评测 runner 绑定旧 Agent API 与 trace 格式，未复制为可执行脚本。新 harness 实现后再适配输入与评分，避免将 501 接口当成已接通的评测。旧 chunk ID 需映射到新来源或商品 ID。
