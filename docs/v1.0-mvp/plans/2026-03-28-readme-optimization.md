# Plan: README.md Optimization (Issue #28)

> **Issue**: #28
> **Created**: 2026-03-28
> **Status**: completed

## Goal

将 README.md 从 282 行精简为 ~80-120 行的人类友好项目名片，去除与 agent.md 的重复内容。

## 问题分析

| 重复内容 | README 当前行数 | agent.md 章节 | 处理方式 |
|---------|---------------|-------------|---------|
| Issue Development Workflow | ~180 行 | §7 | 删除，链接到 agent.md |
| Documentation Structure | ~20 行 | §6 | 精简为 3-5 行概览 |
| Commands | ~15 行 | §9 | 删除，链接到 agent.md |
| Useful Links | ~8 行 | §2 知识索引 | 合并到文档导航 |

## 新 README 结构

```
1. 标题 + 描述 + 徽章              (~3 行)
2. 项目简介                        (~5 行)
3. 特性亮点                        (~8 行)
4. 架构概览（表格）                 (~10 行)
5. 快速开始                        (~20 行)
   - 前置条件
   - 安装 & 运行
6. 文档导航（链接表）               (~10 行)
7. 贡献指南                        (~10 行)
8. License                         (~5 行)
```

## 执行步骤

- [x] Step 1: 创建 Issue #28
- [x] Step 2: 创建 Issue Summary
- [x] Step 3: 创建执行计划（本文件）
- [x] Step 4: 重写 README.md（282 行 → 117 行）
- [x] Step 5: 验证 agent.md 无需修改
- [ ] Step 6: 提交 PR

## 约束

- README 读者是**人类开发者**，不是 AI
- 信息只出现一次：README 定义人类内容，agent.md 定义 AI 内容
- 不改动 agent.md
