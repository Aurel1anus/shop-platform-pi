# Agent（待实现）

src 下的 TypeScript 和 Prompt 文件均为空占位，没有安装 pi，没有运行入口。

- harness：一次请求的上下文、权限、预算、取消、持久化与事件。
- tools：工具 schema 与业务 API 适配，不直接操作业务数据库。
- clients/commerce.ts：调用 Go API，使用当前用户的受控 Bearer 凭证。
- retrieval：自写 RAG 入库与检索；Go 仅保存原始知识资料。
- prompts：未来的系统提示词。

Agent 接管会话、完整模型消息、run 和 trace；业务数据库没有旧 Agent 表。
