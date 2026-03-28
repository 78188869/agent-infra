# Issue #28: docs: optimize README.md - remove duplication with agent.md

> **Status**: completed
> **Created**: 2026-03-28
> **PR**: (pending)

## Summary
README.md 与 agent.md 存在大量内容重复，需要精简 README 为面向人类的项目名片。

## Problem
1. Issue Development Workflow 与 agent.md §7 重复（约 180 行）
2. Documentation Structure 与 agent.md §6 几乎一致
3. Commands 与 agent.md §9 重复
4. 缺少项目简介、快速上手指南等人类友好内容
5. 282 行篇幅过长，信息密度低

## Scope
- [x] 分析 README.md 与 agent.md 的重复内容
- [x] 重写 README.md（282 行 → 117 行）
- [x] 验证 agent.md 无需同步修改

## Knowledge References
- docs/knowledge/agent-md-guide.md

## Key Decisions
1. README 面向人类（项目名片），agent.md 面向 AI（操作手册）
2. 信息只出现一次原则：人类内容定义在 README，agent 内容定义在 agent.md，互相链接引用

## Execution Plan
详见 `plans/2026-03-28-readme-optimization.md`
