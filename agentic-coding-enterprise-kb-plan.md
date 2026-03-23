# 企业级 Agentic Coding 知识库建设方案（可落地版）

> 目标：为 AI Coding/Agent 提供**可靠、可审计、权限一致**的上下文，让 Agent 能“找得到、用得对、可追责”。

## 1. 结论先行（推荐方案）

推荐采用 **“代码索引 + 文档知识库 + 混合检索 + 权限透传 + 评估闭环”** 的组合，而不是单纯做一个文档向量库。

- 核心原则：
  - `Code is source of truth`：代码事实优先，文档用于补充意图和流程。
  - `ACL first`：检索阶段就执行权限裁剪，避免“先召回后过滤”导致泄露。
  - `Hybrid retrieval`：关键词/BM25 + 向量 + 符号/图谱检索联合。
  - `Evaluation before scale`：先用任务集评估，再扩大接入范围。

## 2. 目标能力（面向 AI Coding）

知识库要支持下面 6 类高频任务：

1. 新功能开发：定位相关模块、接口契约、历史 ADR、测试模式。
2. 缺陷修复：回溯变更、同类 incident、runbook、回滚策略。
3. 重构：识别调用链、边界约束、不可破坏接口。
4. 代码评审辅助：检查规范、威胁模型、合规基线。
5. 运维脚本与发布：读取 SRE 手册、变更窗口、依赖影响面。
6. 跨团队协作：统一术语、领域模型、Owner 信息。

## 3. 参考架构（四层）

### 3.1 数据层（Data Sources）

必须接入（按优先级）：

1. Git 仓库（代码、README、测试、CI 配置、迁移脚本）
2. 工程文档（架构设计、ADR、接口规范、编码规范）
3. 研发流程资产（Issue/PR、变更记录、故障复盘、Runbook）
4. 服务运行元数据（服务目录、Owner、依赖关系、SLO）

### 3.2 处理层（Ingestion & Normalization）

- 解析：按语言 AST / 符号级切分（函数、类、模块），避免固定字符切分破坏语义。
- 结构化：统一元数据字段（见第 4 节数据模型）。
- 增量更新：按 commit/webhook 增量索引，不做全量重建。
- 质量门禁：去重、失效检测（过期文档）、机密扫描（密钥/PII）。

### 3.3 检索层（Retrieval）

- `R1` 关键词检索（BM25）
- `R2` 向量召回（代码+文档 embedding）
- `R3` 结构化/图谱检索（符号引用、依赖关系）
- `R4` 重排（cross-encoder 或规则重排）
- `R5` 上下文打包（按 token budget 输出“证据包”）

### 3.4 服务层（Serving for Agents）

- 对 Agent 暴露统一 `Context API`：
  - `retrieve(query, task_type, repo_scope, user_identity)`
  - 返回：`snippets + citations + confidence + policy_flags`
- 每次响应都要附可追溯 citation（仓库/文件/提交/文档链接）。
- 低置信度场景强制澄清或拒答（减少幻觉提交）。

## 4. 最小可用数据模型（MVP Schema）

每条知识记录建议包含：

- `doc_id`：唯一 ID
- `source_type`：`code|doc|adr|issue|runbook|api_spec`
- `repo/project/service`
- `path_or_url`
- `language_or_format`
- `owner_team`
- `acl_tags`：权限标签（部门/项目/密级）
- `version_ref`：`commit_sha|doc_version|updated_at`
- `valid_from/valid_to`：有效期
- `embedding_vector`
- `keywords`
- `relations`：`calls|imports|depends_on|supersedes`
- `quality_score`：新鲜度、完整性、验证状态

## 5. 权限与安全（必须内建）

1. 检索前权限过滤：与 Git/文档系统权限一致（避免越权召回）。
2. 输出前 DLP 检查：密钥、凭证、PII、受限数据脱敏。
3. 操作审计：记录“谁在什么任务里看到了哪些上下文”。
4. 提示注入防护：对外部文档做信任分级，禁止“文档指令覆盖系统策略”。
5. 高风险操作双确认：自动改代码、自动发版必须有人审。

## 6. 90 天落地路线图

### 第 0-30 天：MVP（先打通一条线）

1. 选 2-3 个关键仓库 + 1 套架构文档 + 1 套 runbook 作为试点。
2. 建立增量索引流水线（Git webhook + 文档定时同步）。
3. 上线混合检索（BM25+向量）和最小 citation 输出。
4. 做 30 个真实开发任务评测集（来自历史 PR/Issue）。

交付物：
- 可调用 `Context API`（内部）
- 试点任务评测报告 v1
- 权限透传与审计日志 v1

### 第 31-60 天：可靠性增强

1. 接入符号关系/依赖图检索。
2. 增加重排和上下文压缩，提高相关性。
3. 建立知识健康度看板（新鲜度、失效率、覆盖率）。
4. 将 AI 生成建议与真实 merge 结果做闭环对比。

交付物：
- Top-K 命中率和可用率提升报告
- 知识库 SLO（可用性、延迟、权限准确率）

### 第 61-90 天：规模化推广

1. 扩展到更多业务线仓库与文档系统。
2. 建立“标准接入模板”（新仓库一键接入）。
3. 建立治理机制：数据 owner、过期清理、例外审批。
4. 发布企业级 AI Coding 指南（什么时候可自动化，什么时候必须人工）。

交付物：
- 企业知识库运营手册
- 多团队推广复盘和 ROI 报告

## 7. KPI（建议直接写入 OKR）

技术质量 KPI：

1. `Context Precision@5 >= 0.75`
2. `Citation Coverage >= 95%`（回答中可追溯证据覆盖率）
3. `Permission Error Rate <= 0.1%`
4. `Index Freshness`：代码 <= 10 分钟、文档 <= 24 小时

业务效果 KPI：

1. AI 辅助任务首轮通过率提升 >= 20%
2. 平均定位上下文时间下降 >= 30%
3. 因上下文错误导致的回滚/返工下降 >= 25%

## 8. 技术选型建议（两条路径）

### 路径 A（推荐，落地快）

- GitHub Enterprise + GitHub Copilot（代码语境）
- 企业文档知识库（Confluence/SharePoint/Notion 之一）
- 检索层：Azure AI Search / OpenSearch（混合检索）
- 编排层：自建 Context API（统一权限、重排、审计）

适用：追求 3 个月内见效，有成熟 GitHub 体系。

### 路径 B（更可控）

- 自建索引与检索（OpenSearch + pgvector/向量库）
- 自建 Agent 网关与策略中心
- 可选接入 GraphRAG 做复杂依赖问答

适用：强合规、数据主权要求高、可投入平台工程团队。

## 9. 常见失败模式（提前规避）

1. 只做向量库，不做权限透传。
2. 只接文档，不接代码与变更历史。
3. 不做增量更新，导致“旧知识污染”。
4. 没有评测集，只看主观“感觉变聪明”。
5. 没有 citation 和审计，无法在企业内大规模放开。

## 10. 你们可以直接执行的“下周计划”

1. 选定试点范围：2-3 个仓库 + 1 条核心业务链路。
2. 明确数据 owner：研发负责人、架构 owner、平台 owner。
3. 建立 30 条金标准任务（真实历史任务）。
4. 上线 MVP 检索服务并接入 1 个 coding agent。
5. 每周复盘：命中率、误召回、越权风险、开发者满意度。

---

## 调研依据（官方/一手资料）

- OpenAI Cookbook: RAG with Responses API + Pinecone（检索增强实现参考）  
  https://cookbook.openai.com/examples/responses_api/responses_api_tool_orchestration
- OpenAI Cookbook: Embeddings & retrieval patterns（向量检索基础）  
  https://cookbook.openai.com/
- GitHub Docs: Copilot coding agent（Agent 在代码库上的工作方式）  
  https://docs.github.com/en/copilot/how-tos/agents/copilot-coding-agent
- GitHub Changelog: Knowledge Bases retired, replaced by Spaces（知识能力演进，2025-11-01）  
  https://github.blog/changelog/2025-07-20-knowledge-bases-and-skillsets-are-being-replaced-by-spaces-on-november-1-2025/
- Microsoft Learn: Advanced RAG in Azure AI Search（企业 RAG 设计）  
  https://learn.microsoft.com/en-us/azure/search/retrieval-augmented-generation-overview
- AWS Prescriptive Guidance: best practices for RAG（工程实践）  
  https://docs.aws.amazon.com/prescriptive-guidance/latest/retrieval-augmented-generation-options/rag-best-practices.html
- NIST AI RMF 1.0（治理与风险管理基线）  
  https://www.nist.gov/itl/ai-risk-management-framework
- OWASP Top 10 for LLM Applications（LLM 安全风险）  
  https://owasp.org/www-project-top-10-for-large-language-model-applications/
- Microsoft Research GraphRAG（图增强检索思路）  
  https://www.microsoft.com/en-us/research/project/graphrag/
- SWE-bench（代码 Agent 任务评测基准）  
  https://www.swebench.com/

