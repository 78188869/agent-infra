# TRD：Agentic Coding Platform MVP 技术设计文档

> **版本**：v1.0-draft
> **日期**：2026-03-21
> **状态**：草稿
> **关联文档**：[PRD.md](../PRD.md) | [BRD.md](../BRD.md) | [技术决策记录](./2026-03-21-mvp-technical-decisions.md)

---

## 文档修订历史

| 版本 | 日期 | 修订内容 | 作者 |
|------|------|---------|------|
| v1.0-draft | 2026-03-21 | 初稿：MVP技术设计 | - |

---

## 目录

1. [概述](#1-概述)
2. [技术选型](#2-技术选型)
3. [系统架构设计](#3-系统架构设计)
4. [控制面服务设计](#4-控制面服务设计)
5. [沙箱执行环境设计](#5-沙箱执行环境设计)
6. [数据模型设计](#6-数据模型设计)
7. [API设计](#7-api设计)
8. [部署架构设计](#8-部署架构设计)
9. [监控告警设计](#9-监控告警设计)
10. [安全设计](#10-安全设计)
11. [构建与部署](#11-构建与部署)
12. [测试策略](#12-测试策略)
13. [附录](#13-附录)

---

## 1. 概述

### 1.1 文档目的

本文档定义 Agentic Coding Platform MVP 阶段的技术实现方案，为开发团队提供详细的技术设计指导。

### 1.2 适用范围

本文档适用于 MVP（v1.0）版本，涵盖以下核心功能：
- 任务模板管理
- 任务执行引擎
- 沙箱环境管理
- 人工干预机制
- 能力管理
- 用户面板与管理面板

### 1.3 设计原则

| 原则 | 说明 |
|------|------|
| 模块化单体 | MVP采用单体架构，模块化设计，预留微服务拆分接口 |
| 复用优先 | 复用Claude Code CLI现有能力，不自创配置格式 |
| 快速验证 | 1-2个月交付，优先核心链路可用 |
| 渐进增强 | 预留扩展接口，支持后续迭代增强 |

### 1.4 术语定义

| 术语 | 说明 |
|------|------|
| 控制面 | 管理任务生命周期的服务层 |
| 执行面 | 实际运行任务的沙箱环境 |
| 沙箱 | 任务运行的隔离容器环境 |
| Wrapper | 封装Claude Code CLI的脚本，负责输入输出和状态上报 |
| 调度器 | 负责任务排队、限流、抢占的组件 |
| 执行器 | 负责创建和管理沙箱Pod的组件 |

### 1.5 MVP用户故事覆盖

> 追溯至 PRD 第3章 用户故事

| 用户故事 | 功能模块 | TRD章节 | MVP状态 |
|---------|---------|---------|--------|
| US-D01a 基于模板创建任务 | 任务管理、模板管理 | 4.1, 7.2 | ✓ |
| US-D01b 任务调试 | 任务执行、人工干预 | 4.3, 7.2 | ✓ |
| US-D02 任务执行监控 | 任务执行、日志采集 | 5, 9 | ✓ |
| US-D03 人工干预任务 | 人工干预机制 | 7.2 | ✓ |
| US-A01 租户管理 | 租户管理 | 6, 7.4 | ✓ |
| US-A02 任务模板管理 | 模板管理 | 4.1, 7.3 | ✓ |
| US-A03 能力注册管理 | 能力管理 | 4.1, 7.6 | ✓ |
| US-O01 任务监控 | 监控告警 | 9 | ✓ |
| US-O02 异常处理 | 人工干预机制 | 7.2 | ✓ |

**MVP不包含的用户故事**（后续版本）：

| 用户故事 | 计划版本 | 说明 |
|---------|---------|------|
| US-D01 对话框式创建任务 | v1.1 | 自然语言交互创建 |
| US-D04 MR审核验收 | v1.1 | CI/CD集成后支持 |
| US-D05 知识标注 | v1.2 | 知识沉淀功能 |

---

## 2. 技术选型

### 2.1 技术栈总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                          技术栈总览                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  前端层                                                       │   │
│  │  React 18 + TypeScript 5 + Ant Design 5 + Vite               │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  API网关层                                                    │   │
│  │  阿里云 MSE (路由转发、SSL终结)                                │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  应用层 (控制面服务)                                          │   │
│  │  Go 1.22 + Gin 1.9 + GORM                                    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  执行层 (沙箱环境)                                            │   │
│  │  Docker + Claude Code CLI + Shell Wrapper                    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  基础设施层                                                   │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐ │   │
│  │  │ OceanBase  │ │   Redis    │ │    SLS     │ │    OSS     │ │   │
│  │  │  (数据库)  │ │ (缓存/队列)│ │  (日志)    │ │ (文件存储) │ │   │
│  │  └────────────┘ └────────────┘ └────────────┘ └────────────┘ │   │
│  │  ┌────────────┐ ┌────────────┐                               │   │
│  │  │    ACK     │ │  京东行云  │                               │   │
│  │  │  (K8s集群) │ │  (Git仓库) │                               │   │
│  │  └────────────┘ └────────────┘                               │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 技术选型明细

| 层级 | 组件 | 版本 | 选型理由 |
|------|------|------|---------|
| **前端** | React | 18.x | 生态成熟，组件库丰富 |
| | TypeScript | 5.x | 类型安全，提升代码质量 |
| | Ant Design | 5.x | 企业级UI组件库，开发效率高 |
| | Vite | 5.x | 构建速度快，开发体验好 |
| **API网关** | 阿里云MSE | - | 托管服务，运维成本低 |
| **后端** | Go | 1.22 | 高性能，云原生亲和 |
| | Gin | 1.9.x | 轻量级Web框架，性能优异 |
| | GORM | 1.25.x | ORM库，简化数据库操作 |
| **执行层** | Docker | 24.x | 容器运行时 |
| | Claude Code CLI | latest | Agent运行时 |
| **数据库** | OceanBase | 4.x | MySQL兼容，分布式能力 |
| **缓存** | Redis | 6.x | 高性能缓存，支持多种数据结构 |
| **日志** | 阿里云SLS | - | 托管日志服务，查询分析能力强 |
| **文件存储** | 阿里云OSS | - | 对象存储，持久化文件产物 |
| **容器编排** | 阿里云ACK | 1.24+ | 托管K8s，运维简便 |
| **Git仓库** | 京东行云 | - | 企业内部Git平台 |

### 2.3 开发工具

| 工具 | 用途 |
|------|------|
| Git | 版本控制 |
| Docker | 容器构建 |
| Make | 构建脚本 |
| golangci-lint | Go代码检查 |
| ESLint | 前端代码检查 |
| Postman | API测试 |

---

## 3. 系统架构设计

### 3.1 架构设计理念

#### 3.1.1 核心设计原则

| 原则 | 说明 | 实践方式 |
|------|------|---------|
| **模块化单体** | MVP采用单体架构降低复杂度，但内部模块化设计 | 按职责划分模块，通过接口解耦，预留微服务拆分边界 |
| **控制面与执行面分离** | 控制逻辑与执行环境隔离，各自独立演进 | 控制面无状态设计，执行面按需扩缩容 |
| **异步优先** | 核心链路采用异步处理，提升吞吐和可靠性 | 任务调度、状态同步均采用异步队列 |
| **故障隔离** | 单个任务失败不影响其他任务和系统稳定性 | 每个任务独立沙箱Pod，资源隔离 |
| **可观测优先** | 从设计阶段内置可观测能力 | 结构化日志、指标埋点、链路追踪 |

#### 3.1.2 架构风格选择

```
┌─────────────────────────────────────────────────────────────────────┐
│                      架构风格决策                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  选择：模块化单体 + 分层架构                                         │
│                                                                      │
│  理由：                                                              │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │ 1. MVP阶段（1-2个月）需要快速验证，微服务会拖慢交付         │     │
│  │ 2. 初期流量不大，单体能满足性能需求                          │     │
│  │ 3. 模块化设计使得后续拆分成本低                              │     │
│  │ 4. 简化部署和调试，降低运维复杂度                            │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  未来演进路径：                                                      │
│  MVP(单体) → v1.2(拆分调度服务) → v2.0(全面微服务)                  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                          MVP 技术架构                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    展现层 (Presentation)                      │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │              前端应用 (React SPA)                        │ │   │
│  │  │  ┌─────────────────┐    ┌─────────────────┐            │ │   │
│  │  │  │    用户面板     │    │    管理面板     │            │ │   │
│  │  │  │   /user/*       │    │   /admin/*      │            │ │   │
│  │  │  │  • 任务管理     │    │  • 租户管理     │            │ │   │
│  │  │  │  • 模板浏览     │    │  • 用户管理     │            │ │   │
│  │  │  │  • 执行监控     │    │  • 能力管理     │            │ │   │
│  │  │  └─────────────────┘    └─────────────────┘            │ │   │
│  │  └─────────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                │                                     │
│                          HTTPS/API调用                               │
│                                ▼                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    网关层 (Gateway)                          │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │              阿里云 MSE (托管API网关)                    │ │   │
│  │  │  • 路由转发：/api/* → 控制面服务                        │ │   │
│  │  │  • SSL终结：HTTPS → HTTP                               │ │   │
│  │  │  • 访问日志：请求/响应日志记录                          │ │   │
│  │  │  • 流量控制：(后续扩展)                                 │ │   │
│  │  └─────────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                │                                     │
│                           HTTP (内网)                                │
│                                ▼                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                  应用层 (Application)                        │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │              控制面服务 (Go + Gin)                       │ │   │
│  │  │                                                          │ │   │
│  │  │  ┌────────────────────────────────────────────────────┐ │ │   │
│  │  │  │              API Handler 层                         │ │ │   │
│  │  │  │  职责：HTTP请求处理、参数校验、响应序列化          │ │ │   │
│  │  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────┐ │ │ │   │
│  │  │  │  │ TaskAPI  │ │TemplateAPI│ │ TenantAPI│ │UserAPI│ │ │ │   │
│  │  │  │  └──────────┘ └──────────┘ └──────────┘ └───────┘ │ │ │   │
│  │  │  └────────────────────────────────────────────────────┘ │ │   │
│  │  │                          │                               │ │   │
│  │  │  ┌────────────────────────────────────────────────────┐ │ │   │
│  │  │  │              Service 层 (业务逻辑)                  │ │ │   │
│  │  │  │  职责：业务规则、事务管理、领域模型操作            │ │ │   │
│  │  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────┐ │ │ │   │
│  │  │  │  │ TaskSvc  │ │TemplateSvc│ │ TenantSvc│ │UserSvc│ │ │ │   │
│  │  │  │  └──────────┘ └──────────┘ └──────────┘ └───────┘ │ │ │   │
│  │  │  └────────────────────────────────────────────────────┘ │ │   │
│  │  │                          │                               │ │   │
│  │  │  ┌────────────────────────────────────────────────────┐ │ │   │
│  │  │  │              核心引擎层 (Core Engine)               │ │ │   │
│  │  │  │  职责：核心调度逻辑、执行管理、状态机              │ │ │   │
│  │  │  │  ┌──────────────────┐   ┌──────────────────┐      │ │ │   │
│  │  │  │  │  Task Scheduler  │   │  Task Executor   │      │ │ │   │
│  │  │  │  │  • 优先级队列    │   │  • Pod生命周期   │      │ │ │   │
│  │  │  │  │  • 限流控制      │   │  • 状态同步      │      │ │ │   │
│  │  │  │  │  • 抢占调度      │   │  • 心跳检测      │      │ │ │   │
│  │  │  │  └──────────────────┘   └──────────────────┘      │ │ │   │
│  │  │  └────────────────────────────────────────────────────┘ │ │   │
│  │  │                          │                               │ │   │
│  │  │  ┌────────────────────────────────────────────────────┐ │ │   │
│  │  │  │              基础设施层 (Infrastructure)            │ │ │   │
│  │  │  │  职责：外部资源访问、技术细节封装                  │ │ │   │
│  │  │  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────────┐  │ │ │   │
│  │  │  │  │DBRepo  │ │Redis   │ │SLS     │ │K8s Client  │  │ │ │   │
│  │  │  │  │(GORM)  │ │Client  │ │Client  │ │(client-go) │  │ │ │   │
│  │  │  │  └────────┘ └────────┘ └────────┘ └────────────┘  │ │ │   │
│  │  │  └────────────────────────────────────────────────────┘ │ │   │
│  │  └─────────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                │                                     │
│         ┌──────────────────────┼──────────────────────┐             │
│         ▼                      ▼                      ▼             │
│  ┌─────────────┐       ┌─────────────┐       ┌─────────────┐       │
│  │  数据层 (Data)       │             │                     │       │
│  │  OceanBase  │       │    Redis    │       │  阿里云SLS  │       │
│  │   元数据    │       │ 队列/缓存   │       │   日志      │       │
│  └─────────────┘       └─────────────┘       └─────────────┘       │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    执行层 (Execution)                        │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │              K8s沙箱Pod (动态创建/销毁)                  │ │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │ │   │
│  │  │  │  沙箱Pod 1  │  │  沙箱Pod 2  │  │  沙箱Pod N  │     │ │   │
│  │  │  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │     │ │   │
│  │  │  │ │Claude   │ │  │ │Claude   │ │  │ │Claude   │ │     │ │   │
│  │  │  │ │Code CLI │ │  │ │Code CLI │ │  │ │Code CLI │ │     │ │   │
│  │  │  │ │+Wrapper │ │  │ │+Wrapper │ │  │ │+Wrapper │ │     │ │   │
│  │  │  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │     │ │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘     │ │   │
│  │  └─────────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                   外部集成层 (Integration)                   │   │
│  │  ┌─────────────┐              ┌─────────────┐                 │   │
│  │  │  京东行云   │              │  阿里云OSS  │                 │   │
│  │  │ (Git仓库)   │              │ (文件存储)  │                 │   │
│  │  └─────────────┘              └─────────────┘                 │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.3 分层架构详解

#### 3.3.1 层次职责划分

| 层次 | 职责 | 关键约束 | 技术选型 |
|------|------|---------|---------|
| **展现层** | 用户界面渲染、交互处理 | 不包含业务逻辑 | React + TypeScript |
| **网关层** | 路由、SSL、日志 | 不做业务处理 | 阿里云MSE |
| **应用层** | 业务逻辑处理、流程编排 | 通过接口访问下层 | Go + Gin |
| **数据层** | 数据持久化、缓存 | 被动响应请求 | OceanBase + Redis |
| **执行层** | 任务实际执行 | 与控制面通过HTTP通信 | K8s + Docker |
| **集成层** | 外部系统对接 | 适配器模式隔离变化 | SDK/API |

#### 3.3.2 层间交互规则

```
┌─────────────────────────────────────────────────────────────────────┐
│                        层间交互规则                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  允许的依赖方向（自上而下）：                                        │
│                                                                      │
│  展现层 ──▶ 网关层 ──▶ 应用层 ──▶ 数据层/执行层/集成层              │
│                                                                      │
│  禁止的依赖方向：                                                    │
│                                                                      │
│  ✗ 数据层不能直接调用应用层                                         │
│  ✗ 执行层不能直接访问数据库（通过API）                               │
│  ✗ 展现层不能直接访问数据层（必须通过API）                           │
│                                                                      │
│  特殊交互（回调）：                                                  │
│                                                                      │
│  执行层 ──▶ 应用层 (HTTP回调：状态上报、心跳、完成通知)             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.4 核心数据流

#### 3.4.1 任务创建流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        任务创建数据流                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. 用户操作                                                        │
│     用户面板 ──▶ POST /api/v1/tasks ──▶ MSE网关                     │
│                                                                      │
│  2. 请求处理                                                        │
│     MSE网关 ──▶ 路由转发 ──▶ 控制面服务                             │
│                                                                      │
│  3. 认证校验                                                        │
│     Auth中间件 ──▶ 验证API Key ──▶ 注入用户上下文                   │
│                                                                      │
│  4. 业务处理                                                        │
│     TaskAPI ──▶ TaskSvc ──▶ 参数校验/模板解析                       │
│                │                                                     │
│                ├──▶ 写入OceanBase (任务记录)                        │
│                └──▶ 写入Redis (入优先级队列)                        │
│                                                                      │
│  5. 响应返回                                                        │
│     TaskSvc ──▶ TaskAPI ──▶ JSON响应 ──▶ 用户面板                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3.4.2 任务执行流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        任务执行数据流                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. 任务调度                                                        │
│     Scheduler ──▶ 从Redis队列取出任务                               │
│                 ──▶ 限流检查 (租户配额/全局并发)                    │
│                 ──▶ 抢占判断                                        │
│                 ──▶ 交给Executor                                    │
│                                                                      │
│  2. 环境准备                                                        │
│     Executor ──▶ 解析任务配置                                       │
│               ──▶ 准备CLAUDE.md和启动参数                           │
│               ──▶ 调用K8s API创建沙箱Pod                            │
│               ──▶ 更新任务状态为Running                             │
│                                                                      │
│  3. 任务执行                                                        │
│     沙箱Pod ──▶ Wrapper启动Claude Code CLI                          │
│             ──▶ CLI执行任务                                         │
│             ──▶ Wrapper解析输出                                     │
│             ──▶ 状态上报到控制面                                    │
│             ──▶ Log Agent采集日志到SLS                              │
│                                                                      │
│  4. 状态同步                                                        │
│     Wrapper ──▶ POST /internal/tasks/:id/events                     │
│             ──▶ 控制面更新数据库                                    │
│             ──▶ WebSocket推送前端 (实时更新)                        │
│                                                                      │
│  5. 执行完成                                                        │
│     Wrapper ──▶ POST /internal/tasks/:id/complete                   │
│             ──▶ 控制面收集结果                                      │
│             ──▶ 更新任务状态                                        │
│             ──▶ 清理沙箱Pod                                         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3.4.3 人工干预流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        人工干预数据流                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. 干预请求                                                        │
│     用户面板 ──▶ POST /api/v1/tasks/:id/inject                       │
│             ──▶ MSE网关 ──▶ 控制面                                  │
│                                                                      │
│  2. 干预处理                                                        │
│     TaskSvc ──▶ 验证任务状态                                        │
│             ──▶ 记录干预到数据库                                    │
│             ──▶ 获取沙箱Pod地址                                     │
│             ──▶ 转发干预指令到沙箱                                  │
│                                                                      │
│  3. 沙箱执行                                                        │
│     Wrapper ──▶ 接收干预指令                                        │
│             ──▶ 注入到CLI                                           │
│             ──▶ 继续执行                                            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.5 组件职责详解

| 组件 | 层次 | 职责 | 依赖 | 扩展点 |
|------|------|------|------|--------|
| **用户面板** | 展现层 | 开发者任务操作界面 | API网关 | 可拆分为独立微前端 |
| **管理面板** | 展现层 | 管理员运维界面 | API网关 | 可拆分为独立微前端 |
| **MSE网关** | 网关层 | 路由、SSL、日志 | 无 | 可扩展流量控制 |
| **TaskAPI** | 应用层 | 任务HTTP接口 | TaskSvc | - |
| **TaskSvc** | 应用层 | 任务业务逻辑 | Scheduler, Executor, DB | 可拆分为独立服务 |
| **Task Scheduler** | 应用层 | 任务调度引擎 | Redis, DB | 可替换调度算法 |
| **Task Executor** | 应用层 | 执行管理 | K8s, DB | 可支持多种执行器 |
| **沙箱Pod** | 执行层 | 任务执行环境 | 控制面(API) | 可替换Agent运行时 |

### 3.6 架构扩展性设计

#### 3.6.1 微服务拆分路径

```
┌─────────────────────────────────────────────────────────────────────┐
│                      微服务拆分演进路径                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  MVP (单体)                                                         │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │                    控制面服务 (单体)                        │     │
│  │  TaskAPI │ TemplateSvc │ Scheduler │ Executor │ ...        │     │
│  └────────────────────────────────────────────────────────────┘     │
│                            │                                         │
│                            ▼                                         │
│  v1.2 (拆分调度服务)                                                │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │  API服务     │───▶│  调度服务    │───▶│  执行服务    │          │
│  │  TaskAPI    │    │  Scheduler   │    │  Executor    │          │
│  │  TemplateSvc│    │  Queue       │    │  PodManager  │          │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
│                            │                                         │
│                            ▼                                         │
│  v2.0 (全面微服务)                                                  │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐           │
│  │TaskSvc │ │Template│ │Scheduler│ │Executor│ │Notify  │           │
│  │        │ │Svc     │ │        │ │        │ │Svc     │           │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘           │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3.6.2 预留扩展接口

| 扩展点 | 接口设计 | 当前实现 | 未来扩展 |
|--------|---------|---------|---------|
| **Agent运行时** | `AgentRuntime` 接口 | Claude Code CLI | OpenAI Agent, 自研Agent |
| **调度算法** | `Scheduler` 接口 | 优先级队列+限流 | 公平调度、资源感知调度 |
| **执行器** | `Executor` 接口 | K8s Pod | Kata Container, VM |
| **认证适配器** | `AuthAdapter` 接口 | API Key | 企业IAM, OAuth2 |
| **存储适配器** | `StorageAdapter` 接口 | OceanBase | 其他数据库 |

### 3.7 架构约束与边界

#### 3.7.1 设计约束

| 约束类型 | 约束说明 | 理由 |
|---------|---------|------|
| **技术约束** | 后端必须使用Go语言 | 团队技术栈、云原生生态 |
| **部署约束** | 必须部署在阿里云ACK | 企业基础设施标准 |
| **数据约束** | 元数据必须存储在OceanBase | 企业数据库标准 |
| **安全约束** | 沙箱必须容器隔离 | 安全合规要求 |
| **网络约束** | 沙箱只能访问控制面和Git仓库 | 安全隔离要求 |

#### 3.7.2 架构边界

```
┌─────────────────────────────────────────────────────────────────────┐
│                        架构边界定义                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  平台内部边界：                                                      │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │  控制面服务                                                │     │
│  │  • 负责任务全生命周期管理                                  │     │
│  │  • 不直接执行用户代码                                      │     │
│  │  • 通过API与执行面通信                                     │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │  执行面 (沙箱Pod)                                          │     │
│  │  • 负责任务实际执行                                        │     │
│  │  • 与控制面通过HTTP通信                                    │     │
│  │  • 执行完成后自动销毁                                      │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  外部系统边界：                                                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐        │
│  │ 京东行云       │  │ 阿里云SLS      │  │ 企业IAM        │        │
│  │ (Git仓库)      │  │ (日志)         │  │ (认证-v1.2)    │        │
│  │ 通过Git协议    │  │ 通过SLS SDK    │  │ 通过OAuth2     │        │
│  └────────────────┘  └────────────────┘  └────────────────┘        │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.8 关键技术决策

| 决策点 | 选择 | 备选方案 | 选择理由 |
|--------|------|---------|---------|
| 架构风格 | 模块化单体 | 微服务 | MVP快速交付，后续可拆分 |
| Agent运行时 | Claude Code CLI | Claude API直接调用 | 复用现有工具链，降低开发成本 |
| 任务调度 | Redis队列+自研调度器 | K8s Job | 支持限流、抢占、排队可视化 |
| 沙箱隔离 | Docker容器 | Kata Container | MVP阶段简化，预留升级接口 |
| 日志存储 | 阿里云SLS | ELK | 托管服务，运维成本低 |

---

## 4. 控制面服务设计

### 4.1 模块划分

```
control-plane/
├── cmd/
│   └── main.go                    # 服务入口
├── internal/
│   ├── api/                       # API Handler层
│   │   ├── handler/               # HTTP处理器
│   │   │   ├── task.go
│   │   │   ├── template.go
│   │   │   ├── tenant.go
│   │   │   ├── user.go
│   │   │   └── capability.go
│   │   ├── middleware/            # 中间件
│   │   │   ├── auth.go            # 认证中间件
│   │   │   ├── ratelimit.go       # 限流中间件
│   │   │   ├── logger.go          # 日志中间件
│   │   │   └── recovery.go        # 异常恢复
│   │   └── router.go              # 路由注册
│   │
│   ├── service/                   # Service层
│   │   ├── task_service.go        # 任务业务逻辑
│   │   ├── template_service.go    # 模板业务逻辑
│   │   ├── tenant_service.go      # 租户业务逻辑
│   │   ├── user_service.go        # 用户业务逻辑
│   │   ├── capability_service.go  # 能力业务逻辑
│   │   ├── audit_service.go       # 审计服务
│   │   └── notify_service.go      # 通知服务
│   │
│   ├── scheduler/                 # 调度引擎
│   │   ├── scheduler.go           # 调度器主逻辑
│   │   ├── queue.go               # 优先级队列管理
│   │   ├── ratelimiter.go         # 限流器
│   │   └── preemption.go          # 抢占逻辑
│   │
│   ├── executor/                  # 执行引擎
│   │   ├── executor.go            # 执行管理器
│   │   ├── pod_manager.go         # K8s Pod生命周期管理
│   │   ├── wrapper_client.go      # Wrapper通信客户端
│   │   └── heartbeat.go           # 心跳检测
│   │
│   ├── model/                     # 数据模型
│   │   ├── tenant.go
│   │   ├── user.go
│   │   ├── task.go
│   │   ├── template.go
│   │   └── capability.go
│   │
│   └── config/                    # 配置管理
│       └── config.go
│
├── pkg/                           # 公共库
│   ├── errors/                    # 错误定义
│   ├── logger/                    # 日志工具
│   └── utils/                     # 工具函数
│
└── configs/
    └── config.yaml                # 配置文件
```

### 4.1.1 Service层详细设计

#### Service层职责划分

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Service层模块职责                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                     核心业务服务                              │   │
│  │                                                              │   │
│  │  TaskSvc        任务业务逻辑                                 │   │
│  │  ├── 状态管理与流转                                          │   │
│  │  ├── 参数校验与解析                                          │   │
│  │  ├── 任务创建与查询                                          │   │
│  │  └── 干预处理协调                                            │   │
│  │                                                              │   │
│  │  TemplateSvc    模板业务逻辑                                 │   │
│  │  ├── 模板CRUD操作                                            │   │
│  │  ├── 版本控制管理                                            │   │
│  │  ├── 模板校验与渲染                                          │   │
│  │  └── 发布状态管理                                            │   │
│  │                                                              │   │
│  │  TenantSvc      租户业务逻辑                                 │   │
│  │  ├── 租户资源配额管理                                        │   │
│  │  ├── 配额使用统计                                            │   │
│  │  ├── 租户状态管理                                            │   │
│  │  └── 租户创建与配置                                          │   │
│  │                                                              │   │
│  │  UserSvc        用户业务逻辑                                 │   │
│  │  ├── 用户角色权限管理                                        │   │
│  │  ├── API Key管理                                             │   │
│  │  ├── 用户状态管理                                            │   │
│  │  └── 用户创建与查询                                          │   │
│  │                                                              │   │
│  │  CapabilitySvc  能力业务逻辑                                 │   │
│  │  ├── 工具注册与管理                                          │   │
│  │  ├── Skills包管理                                            │   │
│  │  ├── 能力授权控制                                            │   │
│  │  └── 能力状态管理                                            │   │
│  │                                                              │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                     支撑服务                                  │   │
│  │                                                              │   │
│  │  AuditSvc       审计服务                                     │   │
│  │  ├── 操作日志记录                                            │   │
│  │  ├── 审计日志查询                                            │   │
│  │  ├── 安全事件追踪                                            │   │
│  │  └── 合规性检查支持                                          │   │
│  │                                                              │   │
│  │  NotifySvc      通知服务                                     │   │
│  │  ├── 任务状态通知                                            │   │
│  │  ├── 告警通知                                                │   │
│  │  ├── 事件推送 (WebSocket)                                    │   │
│  │  └── Webhook回调                                             │   │
│  │                                                              │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### 核心Service接口定义

**TaskSvc 接口**：

```go
type TaskService interface {
    // 任务创建与查询
    CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error)
    GetTask(ctx context.Context, taskID string) (*Task, error)
    ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, int64, error)

    // 状态管理
    UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, reason string) error

    // 任务操作
    PauseTask(ctx context.Context, taskID string) error
    ResumeTask(ctx context.Context, taskID string) error
    CancelTask(ctx context.Context, taskID string, reason string) error

    // 干预处理
    InjectInstruction(ctx context.Context, taskID string, content string) error

    // 参数校验
    ValidateTaskParams(ctx context.Context, templateID string, params map[string]interface{}) error

    // 排队查询
    GetQueuePosition(ctx context.Context, taskID string) (int, error)
}
```

**TemplateSvc 接口**：

```go
type TemplateService interface {
    // 模板CRUD
    CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*Template, error)
    GetTemplate(ctx context.Context, templateID string) (*Template, error)
    ListTemplates(ctx context.Context, filter *TemplateFilter) ([]*Template, int64, error)
    UpdateTemplate(ctx context.Context, templateID string, req *UpdateTemplateRequest) error
    DeleteTemplate(ctx context.Context, templateID string) error

    // 版本管理
    GetTemplateVersions(ctx context.Context, templateID string) ([]*TemplateVersion, error)
    GetTemplateByVersion(ctx context.Context, templateID string, version string) (*Template, error)

    // 发布管理
    PublishTemplate(ctx context.Context, templateID string) error
    DeprecateTemplate(ctx context.Context, templateID string) error

    // 校验
    ValidateTemplate(ctx context.Context, spec *TemplateSpec) error
    RenderTemplate(ctx context.Context, templateID string, params map[string]interface{}) (*ResolvedSpec, error)
}
```

**TenantSvc 接口**：

```go
type TenantService interface {
    // 租户管理
    CreateTenant(ctx context.Context, req *CreateTenantRequest) (*Tenant, error)
    GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
    ListTenants(ctx context.Context) ([]*Tenant, error)
    UpdateTenant(ctx context.Context, tenantID string, req *UpdateTenantRequest) error

    // 配额管理
    SetTenantQuota(ctx context.Context, tenantID string, quota *ResourceQuota) error
    GetTenantQuota(ctx context.Context, tenantID string) (*ResourceQuota, error)

    // 配额使用
    GetQuotaUsage(ctx context.Context, tenantID string) (*QuotaUsage, error)
    CheckQuotaAvailable(ctx context.Context, tenantID string, required *ResourceRequest) (bool, error)

    // 状态管理
    SuspendTenant(ctx context.Context, tenantID string, reason string) error
    ActivateTenant(ctx context.Context, tenantID string) error
}
```

**UserSvc 接口**：

```go
type UserService interface {
    // 用户管理
    CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
    GetUser(ctx context.Context, userID string) (*User, error)
    ListUsers(ctx context.Context, tenantID string) ([]*User, error)
    UpdateUser(ctx context.Context, userID string, req *UpdateUserRequest) error

    // 角色权限
    SetUserRole(ctx context.Context, userID string, role UserRole) error
    CheckPermission(ctx context.Context, userID string, permission string) (bool, error)

    // API Key管理
    CreateAPIKey(ctx context.Context, userID string, name string, expiresIn int64) (*APIKey, error)
    ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error)
    RevokeAPIKey(ctx context.Context, keyID string) error
    ValidateAPIKey(ctx context.Context, keyHash string) (*User, error)

    // 状态管理
    DisableUser(ctx context.Context, userID string) error
    EnableUser(ctx context.Context, userID string) error
}
```

**CapabilitySvc 接口**：

```go
type CapabilityService interface {
    // 能力注册
    RegisterCapability(ctx context.Context, req *RegisterCapabilityRequest) (*Capability, error)
    GetCapability(ctx context.Context, capabilityID string) (*Capability, error)
    ListCapabilities(ctx context.Context, filter *CapabilityFilter) ([]*Capability, error)

    // 能力状态
    ActivateCapability(ctx context.Context, capabilityID string) error
    DeactivateCapability(ctx context.Context, capabilityID string) error

    // 授权管理
    AuthorizeCapability(ctx context.Context, tenantID string, capabilityIDs []string) error
    RevokeCapability(ctx context.Context, tenantID string, capabilityIDs []string) error
    GetAuthorizedCapabilities(ctx context.Context, tenantID string) ([]*Capability, error)

    // 能力校验
    ValidateCapabilityConfig(ctx context.Context, capabilityType string, config map[string]interface{}) error
}
```

**AuditSvc 接口**：

```go
type AuditService interface {
    // 操作日志
    LogOperation(ctx context.Context, entry *AuditEntry) error
    QueryOperations(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, int64, error)

    // 安全事件
    LogSecurityEvent(ctx context.Context, event *SecurityEvent) error

    // 导出
    ExportAuditLogs(ctx context.Context, filter *AuditFilter, format string) ([]byte, error)
}
```

**NotifySvc 接口**：

```go
type NotifyService interface {
    // 任务通知
    NotifyTaskStatusChange(ctx context.Context, taskID string, oldStatus, newStatus TaskStatus) error
    NotifyTaskCompletion(ctx context.Context, taskID string, result *TaskResult) error
    NotifyTaskFailure(ctx context.Context, taskID string, err error) error

    // 告警通知
    SendAlert(ctx context.Context, alert *Alert) error

    // WebSocket推送
    PushToClient(ctx context.Context, userID string, event *WebSocketEvent) error
    BroadcastToTenant(ctx context.Context, tenantID string, event *WebSocketEvent) error

    // Webhook
    TriggerWebhook(ctx context.Context, webhookURL string, payload *WebhookPayload) error
}
```

#### Service层依赖关系

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Service层依赖关系图                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  TaskSvc                                                            │
│  ├── 依赖: TemplateSvc (模板解析)                                   │
│  ├── 依赖: TenantSvc (配额检查)                                     │
│  ├── 依赖: UserSvc (权限校验)                                       │
│  ├── 依赖: CapabilitySvc (能力验证)                                 │
│  ├── 依赖: AuditSvc (操作记录)                                      │
│  └── 依赖: NotifySvc (状态通知)                                     │
│                                                                      │
│  TemplateSvc                                                        │
│  ├── 依赖: CapabilitySvc (能力校验)                                 │
│  └── 依赖: AuditSvc (变更记录)                                      │
│                                                                      │
│  TenantSvc                                                          │
│  ├── 依赖: UserSvc (用户关联)                                       │
│  └── 依赖: AuditSvc (操作记录)                                      │
│                                                                      │
│  UserSvc                                                            │
│  └── 依赖: AuditSvc (操作记录)                                      │
│                                                                      │
│  CapabilitySvc                                                      │
│  └── 依赖: AuditSvc (变更记录)                                      │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 任务调度器

#### 4.2.1 调度流程

```
┌─────────────────────────────────────────────────────────────────┐
│                   Task Scheduler 调度流程                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  任务提交 ──▶ ┌─────────────────────────────────────┐           │
│              │      Redis 优先级队列                │           │
│              │  ┌─────────┐ ┌─────────┐ ┌────────┐ │           │
│              │  │  High   │ │ Normal  │ │  Low   │ │           │
│              │  │ Queue   │ │ Queue   │ │ Queue  │ │           │
│              │  └─────────┘ └─────────┘ └────────┘ │           │
│              └─────────────────────────────────────┘           │
│                          │                                      │
│                          ▼                                      │
│              ┌─────────────────────────────────────┐           │
│              │          限流器 (RateLimiter)       │           │
│              │  • 租户级配额检查                    │           │
│              │  • 全局并发控制                      │           │
│              │  • 资源池检查                        │           │
│              └─────────────────────────────────────┘           │
│                          │                                      │
│                          ▼                                      │
│              ┌─────────────────────────────────────┐           │
│              │          调度决策 (Dispatcher)      │           │
│              │  • 抢占判断（高优抢先）              │           │
│              │  • 资源匹配                          │           │
│              │  • 分配沙箱Pod                       │           │
│              └─────────────────────────────────────┘           │
│                          │                                      │
│                          ▼                                      │
│                   交给 Executor 执行                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.2.2 Redis队列设计

| Key模式 | 类型 | 说明 | TTL |
|---------|------|------|-----|
| `queue:tasks:high` | List | 高优先级任务队列（FIFO） | 永久 |
| `queue:tasks:normal` | List | 普通优先级任务队列 | 永久 |
| `queue:tasks:low` | List | 低优先级任务队列 | 永久 |
| `task:{task_id}:meta` | Hash | 任务调度元数据 | 24h |
| `tenant:{tenant_id}:quota:used` | Hash | 租户实时配额使用 | 永久 |
| `lock:scheduler` | String | 调度器分布式锁 | 30s |

#### 4.2.3 限流策略

| 限流维度 | 配置项 | 说明 |
|---------|--------|------|
| 租户级 | TenantMaxConcurrent | 租户最大并发任务数 |
| 租户级 | TenantMaxDailyTasks | 租户每日最大任务数 |
| 全局级 | GlobalMaxConcurrent | 全局最大并发任务数 |
| 算法 | token_bucket | 令牌桶算法 |

#### 4.2.4 抢占逻辑

| 场景 | 抢占条件 | 抢占目标 |
|------|---------|---------|
| 高优先级任务到达 | 高优先级队列有任务，低优先级任务运行中 | 优先抢占Low优先级任务 |
| 资源不足 | 新任务需要资源超过可用资源 | 抢占低优先级任务释放资源 |

### 4.3 任务执行器

#### 4.3.1 执行流程

| 阶段 | 动作 | 说明 |
|------|------|------|
| 1. 接收任务 | 从调度器获取任务 | 任务已通过限流检查 |
| 2. 准备环境 | 生成Pod配置、克隆代码仓库、生成CLAUDE.md | 根据模板配置 |
| 3. 创建Pod | 调用K8s API创建Pod、更新任务状态 | 状态变为Running |
| 4. 监控执行 | 接收状态上报、处理干预、心跳检测 | 实时监控 |
| 5. 处理结果 | 收集产物、更新状态、清理Pod | 任务完成 |

#### 4.3.2 心跳检测机制

| 配置项 | 值 | 说明 |
|--------|-----|------|
| 心跳间隔 | 5s | Wrapper每5s上报心跳 |
| 超时阈值 | 15s | 连续3次未收到心跳视为异常 |
| 异常处理 | 标记任务异常 | 通知运维人员介入 |

---

## 5. 沙箱执行环境设计

### 5.1 设计原则

| 原则 | 说明 |
|------|------|
| 一个容器一个进程 | 符合 K8s 最佳实践，职责分离 |
| Sidecar 模式 | CLI 执行与监控干预分离到不同容器 |
| 共享 PID Namespace | 允许 wrapper 容器向 cli-runner 发送信号 |
| 声明式状态 | 通过共享 Volume 传递状态，避免复杂 IPC |

### 5.2 沙箱Pod架构（Sidecar 模式）

```
┌─────────────────────────────────────────────────────────────────────┐
│                    沙箱Pod (Sidecar 模式)                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  共享配置: shareProcessNamespace: true                              │
│  共享卷: workspace (emptyDir)                                       │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  /workspace/ (共享 Volume - emptyDir)                         │ │
│  │  ├── CLAUDE.md          # Claude Code 项目配置                │ │
│  │  ├── .claude/           # Claude Code 配置目录                │ │
│  │  ├── .mcp.json          # MCP工具配置（可选）                 │ │
│  │  ├── src/               # 业务代码                            │ │
│  │  └── .agent-state/      # 容器间通信状态文件                  │ │
│  │      ├── status.json    # CLI 当前状态                        │ │
│  │      ├── events.jsonl   # 事件流（追加写入）                  │ │
│  │      ├── inject.json    # 待注入指令                          │ │
│  │      └── output/        # 输出产物                            │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌──────────────────────────┐  ┌──────────────────────────┐        │
│  │  主容器: cli-runner      │  │  Sidecar: wrapper        │        │
│  │                          │  │  (Go程序, ~20MB)         │        │
│  │  ┌────────────────────┐  │  │  ┌────────────────────┐  │        │
│  │  │ 启动脚本           │  │  │  │ HTTP Server        │  │        │
│  │  │ 1. 克隆代码仓库    │  │  │  │ :9090              │  │        │
│  │  │ 2. 生成CLAUDE.md   │  │  │  │ • 接收干预指令     │  │        │
│  │  │ 3. 启动CLI         │  │  │  │ • 健康检查         │  │        │
│  │  │ 4. 写入状态文件    │  │  │  └────────────────────┘  │        │
│  │  │ 5. 等待完成        │  │  │  ┌────────────────────┐  │        │
│  │  └────────────────────┘  │  │  │ 心跳服务           │  │        │
│  │                          │  │  │ • 每5秒上报        │  │        │
│  │  Claude Code CLI         │  │  │ • 重试+退避        │  │        │
│  │  (作为子进程运行)        │  │  └────────────────────┘  │        │
│  │                          │  │  ┌────────────────────┐  │        │
│  │  stdout → events.jsonl   │  │  │ 状态监控           │  │        │
│  │  退出码 → status.json    │  │  │ • 监控CLI进程      │  │        │
│  │                          │  │  │ • 读取状态文件     │  │        │
│  │                          │  │  │ • 上报事件         │  │        │
│  │                          │  │  └────────────────────┘  │        │
│  │                          │  │  ┌────────────────────┐  │        │
│  │                          │  │  │ 干预处理           │  │        │
│  │                          │  │  │ • 发送信号         │  │        │
│  │                          │  │  │ • 写入inject.json  │  │        │
│  │                          │  │  └────────────────────┘  │        │
│  └──────────────────────────┘  └──────────────────────────┘        │
│                                                                      │
│  ┌──────────────────────────┐                                       │
│  │  Sidecar: log-agent      │                                       │
│  │  • 采集所有容器日志      │                                       │
│  │  • 实时上报到阿里云SLS   │                                       │
│  │  • 标签: task_id, tenant │                                       │
│  └──────────────────────────┘                                       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.3 容器职责详解

#### 5.3.1 cli-runner（主容器）

| 职责 | 说明 |
|------|------|
| 环境准备 | 克隆 Git 仓库、生成 CLAUDE.md、配置 .mcp.json |
| CLI 执行 | 启动 Claude Code CLI 并传递参数 |
| 状态输出 | 将 CLI 的 stream-json 输出写入 events.jsonl |
| 退出处理 | 将退出码和最终状态写入 status.json |

**启动脚本核心逻辑**：

```bash
#!/bin/bash
# /scripts/cli-runner.sh

set -e

# 1. 克隆代码仓库
git clone "${GIT_REPO_URL}" /workspace/src
cd /workspace/src
git checkout "${GIT_BRANCH}"

# 2. 生成 CLAUDE.md
cat > /workspace/CLAUDE.md <<EOF
${CLAUDE_MD_CONTENT}
EOF

# 3. 启动 CLI 并捕获输出
claude -p "${TASK_PROMPT}" \
       --max-tokens ${MAX_TOKENS} \
       --allowedTools "${ALLOWED_TOOLS}" \
       --output-format stream-json 2>&1 | while read line; do
    echo "$line" >> /workspace/.agent-state/events.jsonl
    # 提取关键状态写入 status.json
    echo "$line" | jq -r 'select(.type == "status")' > /workspace/.agent-state/status.json
done

EXIT_CODE=$?

# 4. 写入最终状态
echo "{\"status\": \"completed\", \"exit_code\": $EXIT_CODE}" > /workspace/.agent-state/status.json

exit $EXIT_CODE
```

#### 5.3.2 wrapper（Sidecar）

| 模块 | 职责 | 说明 |
|------|------|------|
| HTTP Server | 接收干预指令 | 监听 :9090，处理 pause/resume/inject |
| 心跳服务 | 向控制面上报心跳 | 每5秒一次，支持重试+指数退避 |
| 状态监控 | 监控 CLI 进程和状态文件 | 检测进程存活、读取状态变更 |
| 事件上报 | 将 CLI 事件上报到控制面 | 解析 events.jsonl，POST 到控制面 |
| 干预处理 | 执行干预操作 | 发送信号、写入 inject.json |

**wrapper 核心模块**：

```go
// internal/wrapper/wrapper.go
type Wrapper struct {
    controlPlaneURL string
    taskID          string
    httpServer      *HTTPServer
    heartbeat       *HeartbeatService
    stateMonitor    *StateMonitor
    eventReporter   *EventReporter
}

func (w *Wrapper) Run(ctx context.Context) error {
    // 1. 启动 HTTP 服务（接收干预指令）
    go w.httpServer.Start(":9090")

    // 2. 启动心跳服务
    go w.heartbeat.Start(ctx)

    // 3. 启动状态监控
    go w.stateMonitor.Start(ctx)

    // 4. 启动事件上报
    go w.eventReporter.Start(ctx)

    // 5. 等待 CLI 进程退出
    return w.waitForCLI(ctx)
}
```

### 5.4 容器间通信机制

#### 5.4.1 共享 Volume 通信

```
┌─────────────────────────────────────────────────────────────────────┐
│                    容器间通信：共享 Volume                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  cli-runner 写入:                                                   │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  /workspace/.agent-state/events.jsonl                         │ │
│  │  {"type":"status","status":"running","timestamp":1711017600}  │ │
│  │  {"type":"tool_call","name":"Write","file":"main.go"}         │ │
│  │  {"type":"output","content":"function added"}                 │ │
│  │  ...                                                          │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  wrapper 读取:                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  tail -f /workspace/.agent-state/events.jsonl                │ │
│  │  → 解析每一行 → 上报到控制面                                  │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  wrapper 写入:                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  /workspace/.agent-state/inject.json                         │ │
│  │  {"action":"inject","content":"请检查测试用例"}               │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  cli-runner 读取:                                                   │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  CLI 通过 inotify 监听 inject.json 变化                       │ │
│  │  或通过 wrapper 的 HTTP 接口获取                              │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### 5.4.2 进程信号通信（共享 PID Namespace）

| 信号 | 发送方 | 接收方 | 说明 |
|------|--------|--------|------|
| SIGSTOP | wrapper | cli-runner | 暂停 CLI 执行 |
| SIGCONT | wrapper | cli-runner | 恢复 CLI 执行 |
| SIGTERM | wrapper | cli-runner | 取消任务 |
| SIGTERM | K8s | wrapper | Pod 终止信号 |

**wrapper 发送信号的实现**：

```go
func (w *Wrapper) Pause() error {
    // 找到 cli-runner 容器中的 CLI 进程
    // 由于共享 PID Namespace，可以直接发送信号
    return syscall.Kill(cliPid, syscall.SIGSTOP)
}

func (w *Wrapper) Resume() error {
    return syscall.Kill(cliPid, syscall.SIGCONT)
}

func (w *Wrapper) Cancel() error {
    return syscall.Kill(cliPid, syscall.SIGTERM)
}
```

### 5.5 心跳与状态上报

#### 5.5.1 心跳机制

```
┌──────────────┐                    ┌──────────────┐
│  wrapper     │                    │  控制面      │
│  (Sidecar)   │                    │  Executor    │
├──────────────┤                    ├──────────────┤
│              │  POST /heartbeat   │              │
│              │ ─────────────────▶ │              │
│              │ {"task_id":"xxx"}  │              │
│              │                    │              │
│              │ ◀───────────────── │              │
│              │ {"status":"ok"}    │              │
│              │                    │              │
│  每5秒发送一次  │                    │  超时15秒    │
│  失败重试3次   │                    │  标记异常    │
│  指数退避      │                    │              │
└──────────────┘                    └──────────────┘
```

**心跳服务配置**：

| 配置项 | 值 | 说明 |
|--------|-----|------|
| 心跳间隔 | 5s | 正常情况下每5秒发送一次 |
| 超时阈值 | 15s | 连续3次未收到心跳标记异常 |
| 重试次数 | 3 | 失败后最多重试3次 |
| 退避策略 | 指数退避 | 1s → 2s → 4s |

#### 5.5.2 状态上报格式

**心跳请求**：
```json
{
    "task_id": "task-xxx",
    "status": "running",
    "progress": 45,
    "metrics": {
        "tokens_used": 15000,
        "elapsed_seconds": 120
    },
    "timestamp": 1711017600
}
```

**事件上报**：
```json
{
    "task_id": "task-xxx",
    "event_type": "tool_call",
    "payload": {
        "tool": "Write",
        "file": "src/main.go",
        "lines_added": 50
    },
    "timestamp": 1711017600
}
```

### 5.6 干预机制

#### 5.6.1 干预指令 API

**wrapper HTTP 接口**：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /health | 健康检查 |
| POST | /intervention/pause | 暂停 CLI |
| POST | /intervention/resume | 恢复 CLI |
| POST | /intervention/inject | 注入指令 |
| POST | /intervention/cancel | 取消任务 |
| GET | /status | 获取当前状态 |

#### 5.6.2 干预流程

```
用户 → 控制面 API → Executor → wrapper HTTP → cli-runner
                                              │
                                              ├── pause: SIGSTOP
                                              ├── resume: SIGCONT
                                              ├── cancel: SIGTERM
                                              └── inject: 写入 inject.json
```

**注入指令格式**：
```json
{
    "action": "inject",
    "content": "请检查并修复测试用例",
    "timestamp": 1711017600
}
```

### 5.7 Pod 生命周期管理

#### 5.7.1 正常流程

```
1. K8s 创建 Pod (sandbox-task-xxx)
2. 所有容器同时启动
3. wrapper 开始发送心跳
4. cli-runner 克隆代码 → 执行 CLI → 写入状态
5. wrapper 监控状态 → 上报事件
6. CLI 完成 → cli-runner 退出
7. wrapper 检测到 cli-runner 退出 → 上报完成 → 自己退出
8. Pod 变为 Completed
```

#### 5.7.2 异常处理

| 场景 | 检测方式 | 处理策略 |
|------|---------|---------|
| cli-runner 崩溃 | wrapper 监控进程 | 上报 failed，退出 |
| wrapper 崩溃 | 心跳停止 | 控制面标记异常 |
| 心跳超时 | 控制面检测 | 标记任务异常，通知运维 |
| 资源超限 | K8s OOMKilled | Pod 重启或标记失败 |

#### 5.7.3 优雅关闭

```
1. K8s 发送 SIGTERM 给所有容器
2. wrapper 收到信号：
   a. 停止接收新请求
   b. 发送 SIGTERM 给 CLI 进程
   c. 上报 complete 状态
   d. 等待 CLI 退出（最多30s）
   e. 自己退出
3. cli-runner 收到信号：
   a. 保存当前状态
   b. 清理资源
   c. 退出
```

### 5.8 模板参数 → CLI参数映射

| 模板配置 | 环境变量 | CLI参数 |
|---------|---------|---------|
| `spec.goal` | `TASK_PROMPT` | `-p "${TASK_PROMPT}"` |
| `spec.context.initialContext` | `CLAUDE_MD_CONTENT` | 注入到 CLAUDE.md |
| `execution.resources.tokenLimit` | `MAX_TOKENS` | `--max-tokens` |
| `execution.capabilities.tools` | `ALLOWED_TOOLS` | `--allowedTools` |
| `execution.timeout` | `TASK_TIMEOUT` | 脚本控制超时 |
| `spec.context.repo` | `GIT_REPO_URL` | git clone |
| `spec.context.branch` | `GIT_BRANCH` | git checkout |

### 5.9 镜像规划

| 镜像 | 用途 | 大小估算 | Dockerfile |
|------|------|---------|------------|
| cli-runner:v1.0.0 | 主容器，执行 CLI | ~500MB | Dockerfile.cli-runner |
| agent-wrapper:v1.0.0 | Sidecar，监控干预 | ~20MB | Dockerfile.wrapper |
| log-agent:v1.0.0 | Sidecar，日志采集 | ~100MB | 官方镜像 |

### 5.10 通信协议汇总

| 通信方向 | 协议 | 说明 |
|---------|------|------|
| wrapper → 控制面 | HTTP POST | 心跳、事件上报、完成通知 |
| 控制面 → wrapper | HTTP POST | 干预指令（通过 wrapper HTTP API） |
| cli-runner → wrapper | 共享文件 | events.jsonl、status.json |
| wrapper → cli-runner | 进程信号 | SIGSTOP/SIGCONT/SIGTERM |
| 所有容器 → SLS | SLS API | 日志采集 |

---

## 6. 数据模型设计

### 6.1 实体关系图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          核心实体关系                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Tenant (租户)                                                       │
│      │                                                               │
│      ├── 1:N ──▶ User (用户) ──▶ APIKey (密钥)                      │
│      │                  │                                            │
│      │                  └── 1:N ──▶ Task (任务)                      │
│      │                                  │                            │
│      │                                  ├──▶ ExecutionLog (日志)     │
│      │                                  └──▶ Intervention (干预)     │
│      │                                                               │
│      └── 1:N ──▶ Template (模板)                                     │
│                      │                                               │
│                      └── 1:N ──▶ TemplateVersion (版本)              │
│                                                                      │
│  Capability (能力) ── belongs to ──▶ Tenant (或全局)                 │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.2 表结构定义

#### 6.2.1 tenants（租户表）

```sql
CREATE TABLE tenants (
    id              VARCHAR(36) PRIMARY KEY COMMENT '租户ID',
    name            VARCHAR(128) NOT NULL COMMENT '租户名称',
    description     VARCHAR(512) COMMENT '租户描述',

    -- 资源配额
    quota_cpu               INT DEFAULT 100 COMMENT 'CPU核心数上限',
    quota_memory            BIGINT DEFAULT 200 COMMENT '内存上限(GB)',
    quota_concurrency       INT DEFAULT 50 COMMENT '最大并发任务数',
    quota_daily_tasks       INT DEFAULT 1000 COMMENT '每日任务数上限',
    quota_max_token_per_task BIGINT DEFAULT 500000 COMMENT '单任务Token上限',

    -- 状态
    status          ENUM('active', 'suspended') DEFAULT 'active' COMMENT '状态',

    -- 时间戳
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='租户表';
```

#### 6.2.2 users（用户表）

```sql
CREATE TABLE users (
    id              VARCHAR(36) PRIMARY KEY COMMENT '用户ID',
    tenant_id       VARCHAR(36) NOT NULL COMMENT '租户ID',
    username        VARCHAR(64) NOT NULL COMMENT '用户名',
    display_name    VARCHAR(128) COMMENT '显示名称',
    email           VARCHAR(128) COMMENT '邮箱',

    -- 角色与状态
    role            ENUM('developer', 'admin', 'operator', 'reviewer')
                    DEFAULT 'developer' COMMENT '角色',
    status          ENUM('active', 'disabled') DEFAULT 'active' COMMENT '状态',

    -- 时间戳
    last_login_at   TIMESTAMP NULL COMMENT '最后登录时间',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE KEY uk_tenant_username (tenant_id, username),
    INDEX idx_tenant (tenant_id),
    INDEX idx_status (status),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';
```

#### 6.2.3 api_keys（API密钥表）

```sql
CREATE TABLE api_keys (
    id              VARCHAR(36) PRIMARY KEY COMMENT '密钥ID',
    user_id         VARCHAR(36) NOT NULL COMMENT '用户ID',
    key_hash        VARCHAR(128) NOT NULL COMMENT 'API Key哈希值(SHA256)',
    key_prefix      VARCHAR(8) NOT NULL COMMENT 'Key前缀(用于识别)',
    name            VARCHAR(64) COMMENT '密钥名称',
    description     VARCHAR(256) COMMENT '密钥描述',

    -- 有效期与使用
    expires_at      TIMESTAMP NULL COMMENT '过期时间(NULL表示永不过期)',
    last_used_at    TIMESTAMP NULL COMMENT '最后使用时间',
    usage_count     BIGINT DEFAULT 0 COMMENT '使用次数',

    -- 状态
    status          ENUM('active', 'revoked') DEFAULT 'active' COMMENT '状态',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_user (user_id),
    INDEX idx_prefix (key_prefix),
    INDEX idx_status (status),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API密钥表';
```

#### 6.2.4 templates（任务模板表）

```sql
CREATE TABLE templates (
    id              VARCHAR(36) PRIMARY KEY COMMENT '模板ID',
    tenant_id       VARCHAR(36) NOT NULL COMMENT '租户ID',
    name            VARCHAR(128) NOT NULL COMMENT '模板名称',
    version         VARCHAR(32) DEFAULT '1.0.0' COMMENT '版本号',
    description     TEXT COMMENT '模板描述',

    -- 模板定义
    spec            MEDIUMTEXT NOT NULL COMMENT '模板YAML定义',
    scene_type      ENUM('coding', 'ops', 'analysis', 'content', 'custom')
                    DEFAULT 'coding' COMMENT '场景类型',

    -- 状态
    status          ENUM('draft', 'published', 'deprecated')
                    DEFAULT 'draft' COMMENT '状态',

    -- 审计
    created_by      VARCHAR(36) COMMENT '创建者ID',
    published_at    TIMESTAMP NULL COMMENT '发布时间',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE KEY uk_tenant_name_version (tenant_id, name, version),
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_scene_type (scene_type),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务模板表';
```

#### 6.2.5 tasks（任务表）

```sql
CREATE TABLE tasks (
    id              VARCHAR(36) PRIMARY KEY COMMENT '任务ID',
    tenant_id       VARCHAR(36) NOT NULL COMMENT '租户ID',
    template_id     VARCHAR(36) COMMENT '模板ID',
    creator_id      VARCHAR(36) NOT NULL COMMENT '创建者ID',
    parent_task_id  VARCHAR(36) COMMENT '父任务ID(编排时使用)',

    -- 任务信息
    name            VARCHAR(256) COMMENT '任务名称',
    description     TEXT COMMENT '任务描述',

    -- 状态与进度
    status          ENUM('pending', 'scheduled', 'running', 'paused',
                        'waiting_approval', 'retrying', 'succeeded',
                        'failed', 'cancelled')
                    DEFAULT 'pending' COMMENT '任务状态',
    progress        INT DEFAULT 0 COMMENT '执行进度(0-100)',
    current_stage   VARCHAR(64) COMMENT '当前阶段',

    -- 配置与参数
    params          JSON COMMENT '运行时参数',
    resolved_spec   MEDIUMTEXT COMMENT '参数替换后的完整配置',
    priority        ENUM('high', 'normal', 'low') DEFAULT 'normal' COMMENT '优先级',

    -- 执行信息
    pod_name        VARCHAR(128) COMMENT 'K8s Pod名称',
    sandbox_id      VARCHAR(64) COMMENT '沙箱标识',
    retry_count     INT DEFAULT 0 COMMENT '重试次数',
    max_retries     INT DEFAULT 3 COMMENT '最大重试次数',

    -- 结果与指标
    result          JSON COMMENT '执行结果',
    error_message   TEXT COMMENT '错误信息',
    error_code      VARCHAR(32) COMMENT '错误码',
    metrics         JSON COMMENT '执行指标(tokens/time/resources)',

    -- 时间戳
    scheduled_at    TIMESTAMP NULL COMMENT '调度时间',
    started_at      TIMESTAMP NULL COMMENT '开始时间',
    finished_at     TIMESTAMP NULL COMMENT '完成时间',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_creator (creator_id),
    INDEX idx_template (template_id),
    INDEX idx_status_created (status, created_at),
    INDEX idx_scheduled_at (scheduled_at),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (template_id) REFERENCES templates(id),
    FOREIGN KEY (creator_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务表';
```

#### 6.2.6 execution_logs（执行日志表）

```sql
-- 注：完整日志存储在阿里云SLS，此表仅存储关键事件索引
CREATE TABLE execution_logs (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '日志ID',
    task_id         VARCHAR(36) NOT NULL COMMENT '任务ID',

    -- 事件信息
    event_type      ENUM('status_change', 'tool_call', 'tool_result',
                        'llm_input', 'llm_output', 'error', 'heartbeat',
                        'intervention', 'metric', 'checkpoint')
                    NOT NULL COMMENT '事件类型',
    event_name      VARCHAR(64) COMMENT '事件名称',
    content         JSON COMMENT '事件内容',

    -- 关联信息
    parent_event_id BIGINT COMMENT '父事件ID',

    timestamp       TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '事件时间',

    INDEX idx_task_time (task_id, timestamp),
    INDEX idx_task_event (task_id, event_type),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='执行日志表';
```

#### 6.2.7 interventions（人工干预记录表）

```sql
CREATE TABLE interventions (
    id              VARCHAR(36) PRIMARY KEY COMMENT '干预ID',
    task_id         VARCHAR(36) NOT NULL COMMENT '任务ID',
    operator_id     VARCHAR(36) NOT NULL COMMENT '操作者ID',

    -- 干预信息
    action          ENUM('pause', 'resume', 'cancel', 'inject', 'modify')
                    NOT NULL COMMENT '干预动作',
    content         JSON COMMENT '干预内容',
    reason          VARCHAR(512) COMMENT '干预原因',

    -- 结果
    result          JSON COMMENT '执行结果',
    status          ENUM('pending', 'applied', 'failed')
                    DEFAULT 'pending' COMMENT '状态',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_task (task_id),
    INDEX idx_operator (operator_id),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (task_id) REFERENCES tasks(id),
    FOREIGN KEY (operator_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='人工干预记录表';
```

#### 6.2.8 capabilities（能力注册表）

```sql
CREATE TABLE capabilities (
    id              VARCHAR(36) PRIMARY KEY COMMENT '能力ID',
    tenant_id       VARCHAR(36) COMMENT '租户ID(NULL表示全局能力)',

    -- 能力信息
    type            ENUM('tool', 'skill', 'agent_runtime')
                    NOT NULL COMMENT '能力类型',
    name            VARCHAR(64) NOT NULL COMMENT '能力名称',
    description     TEXT COMMENT '能力描述',
    version         VARCHAR(32) DEFAULT '1.0.0' COMMENT '版本',

    -- 配置
    config          JSON COMMENT '能力配置',
    schema          JSON COMMENT '参数Schema',

    -- 权限
    permission_level ENUM('public', 'restricted', 'admin_only')
                    DEFAULT 'public' COMMENT '权限级别',

    -- 状态
    status          ENUM('active', 'inactive') DEFAULT 'active' COMMENT '状态',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE KEY uk_tenant_type_name (tenant_id, type, name),
    INDEX idx_type_status (type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='能力注册表';
```

#### 6.2.9 knowledge（知识库表）- v1.2+ 预留

> **注意**：此表为MVP阶段预留的数据结构，v1.2版本启用知识沉淀功能时使用。

```sql
CREATE TABLE knowledge (
    id              VARCHAR(36) PRIMARY KEY COMMENT '知识ID',
    tenant_id       VARCHAR(36) NOT NULL COMMENT '租户ID',

    -- 类型与来源
    type            ENUM('failure_case', 'success_case', 'manual_annotation')
                    NOT NULL COMMENT '知识类型',
    source_task_id  VARCHAR(36) COMMENT '来源任务ID',

    -- 内容
    title           VARCHAR(256) COMMENT '知识标题',
    content         TEXT NOT NULL COMMENT '知识内容',
    tags            JSON COMMENT '标签列表',

    -- 标注信息
    annotator_id    VARCHAR(36) COMMENT '标注者ID(人工标注时)',

    -- 状态
    status          ENUM('active', 'archived') DEFAULT 'active' COMMENT '状态',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    INDEX idx_tenant_type (tenant_id, type),
    INDEX idx_source_task (source_task_id),
    INDEX idx_status (status),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (source_task_id) REFERENCES tasks(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识库表(v1.2+)';
```

---

## 7. API设计

### 7.1 API列表

#### 7.1.1 任务管理 `/api/v1/tasks`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| POST | /tasks | 创建任务 | developer |
| GET | /tasks | 任务列表 | developer |
| GET | /tasks/:id | 任务详情 | developer |
| POST | /tasks/:id/pause | 暂停任务 | developer, operator |
| POST | /tasks/:id/resume | 恢复任务 | developer, operator |
| POST | /tasks/:id/cancel | 取消任务 | developer, operator |
| POST | /tasks/:id/inject | 注入指令 | developer, operator |
| POST | /tasks/:id/retry | 重试任务 | developer, operator |
| GET | /tasks/:id/logs | 获取日志 | developer, operator |
| GET | /tasks/:id/queue-position | 查询排队位置 | developer |
| GET | /tasks/:id/metrics | 获取执行指标 | developer |

#### 7.1.2 模板管理 `/api/v1/templates`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /templates | 模板列表 | developer |
| GET | /templates/:id | 模板详情 | developer |
| POST | /templates | 创建模板 | admin |
| PUT | /templates/:id | 更新模板 | admin |
| DELETE | /templates/:id | 删除模板 | admin |
| POST | /templates/:id/publish | 发布模板 | admin |
| POST | /templates/:id/deprecate | 废弃模板 | admin |
| GET | /templates/:id/versions | 版本历史 | admin |
| POST | /templates/:id/validate | 校验模板 | admin |

#### 7.1.3 租户管理 `/api/v1/tenants`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /tenants | 租户列表 | admin |
| GET | /tenants/:id | 租户详情 | admin |
| POST | /tenants | 创建租户 | admin |
| PUT | /tenants/:id | 更新租户 | admin |
| PUT | /tenants/:id/quota | 设置配额 | admin |
| GET | /tenants/:id/usage | 配额使用情况 | admin |
| POST | /tenants/:id/suspend | 暂停租户 | admin |
| POST | /tenants/:id/activate | 激活租户 | admin |

#### 7.1.4 用户管理 `/api/v1/users`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /users | 用户列表 | admin |
| GET | /users/:id | 用户详情 | admin |
| POST | /users | 创建用户 | admin |
| PUT | /users/:id | 更新用户 | admin |
| DELETE | /users/:id | 删除用户 | admin |
| POST | /users/:id/api-keys | 生成API Key | admin |
| GET | /users/:id/api-keys | API Key列表 | admin |
| DELETE | /users/:id/api-keys/:keyId | 删除API Key | admin |

#### 7.1.5 能力管理 `/api/v1/capabilities`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /capabilities | 能力列表 | developer |
| GET | /capabilities/:id | 能力详情 | developer |
| POST | /capabilities | 注册能力 | admin |
| PUT | /capabilities/:id | 更新能力 | admin |
| DELETE | /capabilities/:id | 删除能力 | admin |

#### 7.1.6 监控指标 `/api/v1/metrics`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /metrics/dashboard | Dashboard数据 | operator |
| GET | /metrics/tasks | 任务统计 | operator |
| GET | /metrics/resources | 资源使用 | operator |
| GET | /metrics/tenants | 租户统计 | operator |

#### 7.1.7 内部接口 `/api/v1/internal`

| 方法 | 路径 | 说明 | 调用方 |
|------|------|------|--------|
| POST | /internal/tasks/:id/events | 上报执行事件 | 沙箱Wrapper |
| POST | /internal/tasks/:id/heartbeat | 心跳上报 | 沙箱Wrapper |
| POST | /internal/tasks/:id/complete | 任务完成通知 | 沙箱Wrapper |
| POST | /internal/tasks/:id/intervention | 接收干预指令 | 沙箱Wrapper |

### 7.2 认证授权

#### 7.2.1 认证流程

```
Client → MSE网关(路由转发) → Go服务
                              │
                              ▼
                         Auth Middleware
                         ├── 1. 提取API Key
                         ├── 2. 查Redis缓存
                         ├── 3. 查DB验证
                         └── 4. 注入上下文

Header: Authorization: Bearer <api_key>
```

#### 7.2.2 RBAC权限模型

| 角色 | 权限说明 |
|------|---------|
| developer | 创建/管理自己的任务、查看模板 |
| admin | 管理租户、用户、模板、能力 |
| operator | 监控任务、处理异常、人工干预 |
| reviewer | 审核MR、知识标注（v1.1+） |

### 7.3 错误码

| 错误码范围 | 类别 | 示例 |
|-----------|------|------|
| 0 | 成功 | 0 |
| 40000 | 请求参数错误 | 40001 参数缺失, 40002 参数格式错误 |
| 40100 | 认证错误 | 40100 未认证, 40101 Key无效, 40102 Key过期 |
| 40300 | 权限错误 | 40300 无权限, 40301 角色权限不足 |
| 40400 | 资源不存在 | 40400 任务不存在, 40401 模板不存在 |
| 40900 | 资源冲突 | 40900 资源已存在, 40901 状态冲突 |
| 42900 | 限流错误 | 42900 请求过于频繁, 42901 配额超限 |
| 50000 | 服务内部错误 | 50000 内部错误, 50001 数据库错误 |
| 50300 | 服务不可用 | 50300 资源不足, 50301 服务维护中 |

### 7.4 响应格式

**成功响应**：
```json
{
    "code": 0,
    "message": "success",
    "data": { },
    "request_id": "req-xxx"
}
```

**错误响应**：
```json
{
    "code": 40400,
    "message": "任务不存在",
    "data": null,
    "request_id": "req-xxx"
}
```

**分页响应**：
```json
{
    "code": 0,
    "data": {
        "items": [...],
        "pagination": {
            "page": 1,
            "page_size": 20,
            "total": 100,
            "total_pages": 5
        }
    }
}
```

**分页请求参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码（从1开始） |
| page_size | int | 20 | 每页条数（最大100） |
| sort_by | string | created_at | 排序字段 |
| sort_order | string | desc | 排序方向（asc/desc） |

**示例请求**：
```
GET /api/v1/tasks?page=1&page_size=20&sort_by=created_at&sort_order=desc
```

### 7.5 WebSocket API设计

用于实时推送任务状态变更、日志流、进度更新等事件。

#### 7.5.1 连接端点

| 端点 | 说明 |
|------|------|
| `ws://{host}/api/v1/ws` | WebSocket连接端点 |

**认证方式**：
- 通过查询参数传递Token：`ws://{host}/api/v1/ws?token={api_key}`
- 或通过首次消息认证（连接后立即发送auth消息）

#### 7.5.2 消息格式

**通用消息结构**：
```json
{
    "type": "message_type",
    "timestamp": 1711017600000,
    "payload": { }
}
```

**消息类型定义**：

| 消息类型 | 方向 | 说明 |
|---------|------|------|
| auth | C→S | 客户端认证 |
| auth_result | S→C | 认证结果 |
| subscribe | C→S | 订阅频道 |
| unsubscribe | C→S | 取消订阅 |
| subscribed | S→C | 订阅确认 |
| task_status_change | S→C | 任务状态变更 |
| task_log | S→C | 任务日志流 |
| task_progress | S→C | 任务进度更新 |
| alert | S→C | 告警通知 |
| ping | C→S / S→C | 心跳 |
| pong | S→C / C→S | 心跳响应 |

#### 7.5.3 订阅频道

| 频道 | 说明 | 权限 |
|------|------|------|
| `task:{task_id}` | 单个任务的事件 | 任务创建者 |
| `tenant:{tenant_id}:tasks` | 租户所有任务事件 | 租户管理员 |
| `alerts` | 系统告警 | operator |

**订阅示例**：
```json
// 客户端订阅
{
    "type": "subscribe",
    "channel": "task:abc-123"
}

// 服务端确认
{
    "type": "subscribed",
    "channel": "task:abc-123",
    "timestamp": 1711017600000
}
```

#### 7.5.4 事件消息示例

**任务状态变更**：
```json
{
    "type": "task_status_change",
    "timestamp": 1711017600000,
    "payload": {
        "task_id": "abc-123",
        "old_status": "scheduled",
        "new_status": "running",
        "started_at": "2026-03-21T10:30:00Z"
    }
}
```

**任务日志流**：
```json
{
    "type": "task_log",
    "timestamp": 1711017600000,
    "payload": {
        "task_id": "abc-123",
        "log_level": "info",
        "message": "Starting Claude Code CLI...",
        "sequence": 1
    }
}
```

**任务进度更新**：
```json
{
    "type": "task_progress",
    "timestamp": 1711017600000,
    "payload": {
        "task_id": "abc-123",
        "progress": 45,
        "stage": "coding",
        "metrics": {
            "tokens_used": 15000,
            "elapsed_seconds": 120
        }
    }
}
```

#### 7.5.5 心跳机制

| 配置 | 值 | 说明 |
|------|-----|------|
| 客户端心跳间隔 | 30s | 客户端发送ping |
| 服务端心跳响应 | < 1s | 服务端回复pong |
| 超时断开 | 60s | 无心跳则断开 |
| 重连机制 | 客户端负责 | 指数退避重连 |

**心跳消息**：
```json
// Ping
{"type": "ping", "timestamp": 1711017600000}

// Pong
{"type": "pong", "timestamp": 1711017600000}
```

---

## 8. 部署架构设计

### 8.1 Namespace划分

```
┌─────────────────────────────────────────────────────────────────────┐
│                     ACK K8s集群 Namespace规划                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                control-plane Namespace                       │   │
│  │                                                                │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │  控制面服务 (control-plane)                              │ │   │
│  │  │  ┌───────────┐  ┌───────────┐                          │ │   │
│  │  │  │  Pod 1    │  │  Pod 2    │   (2副本)                │ │   │
│  │  │  └───────────┘  └───────────┘                          │ │   │
│  │  │  Service: control-plane-svc (ClusterIP:8080)            │ │   │
│  │  └─────────────────────────────────────────────────────────┘ │   │
│  │                                                                │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │  前端服务 (frontend)                                     │ │   │
│  │  │  ┌───────────────────────────────────────────────────┐  │ │   │
│  │  │  │  Nginx (用户面板 + 管理面板)                       │  │ │   │
│  │  │  │  /         → 用户面板                              │  │ │   │
│  │  │  │  /admin/*  → 管理面板                              │  │ │   │
│  │  │  │  /api/*    → 代理到控制面服务                      │  │ │   │
│  │  │  └───────────────────────────────────────────────────┘  │ │   │
│  │  │  ┌───────────┐  ┌───────────┐                          │ │   │
│  │  │  │  Pod 1    │  │  Pod 2    │   (2副本)                │ │   │
│  │  │  └───────────┘  └───────────┘                          │ │   │
│  │  │  Service: frontend-svc (ClusterIP:80)                   │ │   │
│  │  └─────────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                     sandbox Namespace                        │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐               │   │
│  │  │ 沙箱Pod-1 │  │ 沙箱Pod-2 │  │ 沙箱Pod-N │   (动态创建)  │   │
│  │  │ (Task-A)  │  │ (Task-B)  │  │ (Task-?)  │               │   │
│  │  └───────────┘  └───────────┘  └───────────┘               │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 8.2 K8s资源规划

| Namespace | 资源类型 | 资源名称 | 副本数 | 配置 |
|-----------|---------|---------|--------|------|
| control-plane | Deployment | control-plane | 2 | 2核4G |
| control-plane | Service | control-plane-svc | - | ClusterIP:8080 |
| control-plane | ConfigMap | app-config | - | 应用配置 |
| control-plane | Secret | db-credentials | - | 数据库凭证 |
| control-plane | Deployment | frontend | 2 | 1核2G |
| control-plane | Service | frontend-svc | - | ClusterIP:80 |
| sandbox | Pod | sandbox-{task-id} | 动态 | 按模板配置 |

### 8.3 资源配额估算

| 资源 | 配置 | 说明 |
|------|------|------|
| 控制面Pod | 2核4G x 2副本 | API + 调度 + 执行管理 |
| 前端Pod | 1核2G x 2副本 | Nginx静态资源服务 |
| 沙箱Pod | 2核4G ~ 4核8G | 按任务模板配置，动态创建 |
| 最大并发沙箱 | 50个 | 初始配额，按需扩展 |
| 总计 | ~100核200G | MVP初始容量 |

### 8.4 网络策略

```
┌─────────────────────────────────────────────────────────────────────┐
│                        网络访问路径                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  外部访问:                                                          │
│  用户 ──▶ MSE网关 ──▶ frontend-svc ──▶ 前端Pod                     │
│                      │                                               │
│                      ▼                                               │
│               API请求 ──▶ control-plane-svc ──▶ 控制面Pod           │
│                                                                      │
│  内部通信:                                                          │
│  控制面Pod ──▶ OceanBase / Redis / SLS / OSS                       │
│  控制面Pod ──▶ K8s API Server ──▶ 创建/管理沙箱Pod                 │
│  沙箱Pod   ──▶ control-plane-svc ──▶ 回调上报                      │
│  沙箱Pod   ──▶ 京东行云 (Git操作)                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 9. 监控告警设计

### 9.1 监控指标

#### 9.1.1 业务指标

| 指标名称 | 类型 | 说明 | 采集方式 |
|---------|------|------|---------|
| task_total | Counter | 任务总数 | 应用埋点 |
| task_status_count | Gauge | 各状态任务数 | 数据库查询 |
| task_duration_seconds | Histogram | 任务执行时长 | 应用埋点 |
| task_queue_size | Gauge | 队列中等待任务数 | Redis查询 |
| task_concurrent_count | Gauge | 并发执行任务数 | K8s查询 |

#### 9.1.2 系统指标

| 指标名称 | 说明 | 来源 |
|---------|------|------|
| cpu_usage | CPU使用率 | K8s Metrics |
| memory_usage | 内存使用率 | K8s Metrics |
| pod_count | Pod数量 | K8s API |
| http_request_duration | API响应时间 | 应用中间件 |
| http_request_count | API请求总数 | 应用中间件 |
| db_connection_pool | 数据库连接池状态 | GORM |

#### 9.1.3 基础设施指标

| 指标名称 | 说明 | 来源 |
|---------|------|------|
| redis_connection_count | Redis连接数 | Redis INFO |
| redis_memory_usage | Redis内存使用 | Redis INFO |
| db_query_duration | 数据库查询时长 | 应用埋点 |
| sls_log_latency | SLS日志延迟 | SLS API |

### 9.2 告警规则

| 告警名称 | 条件 | 级别 | 通知方式 |
|---------|------|------|---------|
| 服务不可用 | 控制面Pod全部不可用 | P0 | 电话 + 短信 |
| API错误率过高 | 5xx错误率 > 5% | P1 | 短信 + 邮件 |
| 任务失败率过高 | 任务失败率 > 20% | P1 | 短信 + 邮件 |
| 沙箱Pod创建失败 | 连续3次创建失败 | P1 | 邮件 |
| Redis连接延迟 | 延迟 > 100ms | P2 | 邮件 |
| 数据库连接池耗尽 | 使用率 > 90% | P1 | 短信 + 邮件 |
| 内存使用过高 | 内存使用率 > 85% | P2 | 邮件 |

---

## 10. 安全设计

### 10.1 认证授权

#### 10.1.1 API Key认证

| 项目 | 设计 |
|------|------|
| Key格式 | `acp_` 前缀 + 32位随机字符 |
| 存储方式 | SHA256哈希后存储 |
| 传输方式 | Header: `Authorization: Bearer <api_key>` |
| 有效期 | 可设置过期时间，默认永不过期 |
| 撤销 | 支持手动撤销 |

#### 10.1.2 RBAC权限模型

| 角色 | 权限范围 |
|------|---------|
| developer | 创建/管理自己的任务、查看模板 |
| admin | 管理租户、用户、模板、能力 |
| operator | 监控任务、处理异常、人工干预 |
| reviewer | 审核MR、知识标注（v1.1+） |

#### 10.1.3 企业IAM对接（预留接口）

```go
// IAM适配器接口（后续实现）
type IAMAdapter interface {
    // 验证用户身份
    Authenticate(token string) (*UserContext, error)
    // 获取用户权限
    GetPermissions(userID string) ([]Permission, error)
}
```

### 10.2 数据安全

#### 10.2.1 敏感数据处理

| 数据类型 | 存储方式 | 访问控制 |
|---------|---------|---------|
| API Key | SHA256哈希 | 仅用户和管理员可管理 |
| 数据库密码 | K8s Secret加密 | 仅应用可访问 |
| Git凭证 | K8s Secret加密 | 仅沙箱Pod可访问 |
| 任务参数 | 明文（敏感参数标记） | 按任务权限控制 |

#### 10.2.2 审计日志

| 审计事件 | 记录内容 |
|---------|---------|
| 用户登录 | 用户ID、时间、IP |
| 任务操作 | 操作类型、任务ID、操作者、时间 |
| 配置变更 | 变更内容、操作者、时间 |
| 权限变更 | 变更内容、操作者、时间 |

### 10.3 网络安全

#### 10.3.1 网络隔离

| 隔离级别 | 说明 |
|---------|------|
| 租户隔离 | 数据库层按tenant_id隔离 |
| 任务隔离 | 每个任务独立沙箱Pod |
| 网络隔离 | 沙箱Pod仅允许访问控制面和外部Git |

#### 10.3.2 传输加密

| 场景 | 加密方式 |
|------|---------|
| 外部访问 | HTTPS (MSE网关) |
| 内部通信 | HTTP (K8s集群内网) |
| 数据库连接 | TLS (可选) |

---

## 11. 构建与部署

### 11.1 项目结构

```
agent-infra/
├── cmd/
│   ├── control-plane/          # 控制面服务入口
│   │   └── main.go
│   └── wrapper/                # Agent Wrapper入口（Sidecar）
│       └── main.go
├── internal/
│   ├── api/                    # API层
│   │   ├── handler/            # HTTP处理器
│   │   ├── middleware/         # 中间件
│   │   └── router.go           # 路由注册
│   ├── service/                # 业务逻辑层
│   ├── scheduler/              # 调度引擎
│   ├── executor/               # 执行引擎
│   ├── model/                  # 数据模型
│   ├── config/                 # 配置管理
│   └── wrapper/                # Agent Wrapper模块
│       ├── heartbeat.go        # 心跳服务
│       ├── state_monitor.go    # 状态文件监控
│       ├── http_server.go      # HTTP服务
│       ├── intervention.go     # 干预处理
│       └── reporter.go         # 事件上报
├── pkg/                        # 公共库
│   ├── errors/                 # 错误定义
│   ├── logger/                 # 日志工具
│   └── utils/                  # 工具函数
├── web/                        # 前端代码
│   ├── user-panel/             # 用户面板
│   └── admin-panel/            # 管理面板
├── deploy/
│   ├── k8s/                    # K8s配置
│   │   ├── control-plane/
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   ├── configmap.yaml
│   │   │   └── secret.yaml
│   │   └── sandbox/
│   │       ├── namespace.yaml
│   │       └── pod-template.yaml
│   └── dockerfiles/
│       ├── Dockerfile.control-plane
│       ├── Dockerfile.frontend
│       ├── Dockerfile.cli-runner    # CLI Runner镜像（主容器）
│       └── Dockerfile.agent-wrapper # Agent Wrapper镜像（Sidecar）
├── scripts/
│   ├── migrate.sh              # 数据库迁移
│   └── cli-runner.sh           # CLI Runner脚本
├── configs/
│   └── config.yaml             # 配置文件示例
├── Makefile
├── go.mod
└── go.sum
```

### 11.2 Dockerfile

#### 11.2.1 控制面服务

```dockerfile
# deploy/dockerfiles/Dockerfile.control-plane

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# 依赖缓存层
COPY go.mod go.sum ./
RUN go mod download

# 构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o control-plane ./cmd/control-plane

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# 安装必要工具
RUN apk --no-cache add ca-certificates tzdata

# 复制二进制
COPY --from=builder /build/control-plane .

# 非root用户
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./control-plane"]
```

#### 11.2.2 前端服务

```dockerfile
# deploy/dockerfiles/Dockerfile.frontend

# Build stage
FROM node:20-alpine AS builder

WORKDIR /build

# 用户面板构建
COPY web/user-panel/package*.json ./user-panel/
WORKDIR /build/user-panel
RUN npm ci
COPY web/user-panel/ .
RUN npm run build

# 管理面板构建
WORKDIR /build
COPY web/admin-panel/package*.json ./admin-panel/
WORKDIR /build/admin-panel
RUN npm ci
COPY web/admin-panel/ .
RUN npm run build

# Runtime stage
FROM nginx:alpine

# 复制构建产物
COPY --from=builder /build/user-panel/dist /usr/share/nginx/html/user
COPY --from=builder /build/admin-panel/dist /usr/share/nginx/html/admin

# 复制nginx配置
COPY deploy/dockerfiles/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

#### 11.2.3 CLI Runner镜像（主容器）

```dockerfile
# deploy/dockerfiles/Dockerfile.cli-runner

FROM ubuntu:22.04

# 安装基础工具
RUN apt-get update && apt-get install -y \
    git \
    curl \
    wget \
    vim \
    jq \
    python3 \
    python3-pip \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

# 安装 Claude Code CLI
RUN curl -fsSL https://claude.ai/install.sh | sh

# 安装 runner 脚本
COPY scripts/cli-runner.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/cli-runner.sh

# 创建工作目录
RUN mkdir -p /workspace && chmod 777 /workspace

WORKDIR /workspace

# 非root用户
RUN useradd -m -u 1000 sandbox
USER sandbox

ENTRYPOINT ["/usr/local/bin/cli-runner.sh"]
```

#### 11.2.4 Agent Wrapper镜像（Sidecar）

```dockerfile
# deploy/dockerfiles/Dockerfile.agent-wrapper

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# 依赖缓存层
COPY go.mod go.sum ./
RUN go mod download

# 构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o wrapper ./cmd/wrapper

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# 安装必要工具
RUN apk --no-cache add ca-certificates

# 复制二进制
COPY --from=builder /build/wrapper .

# 非root用户
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 9090

ENTRYPOINT ["./wrapper"]
```

#### 11.2.5 镜像规划汇总

| 镜像 | 用途 | 大小估算 |
|------|------|---------|
| control-plane | 控制面服务 | ~30MB |
| frontend | 前端服务（Nginx） | ~50MB |
| cli-runner | 沙箱主容器（含Claude Code CLI） | ~500MB |
| agent-wrapper | 沙箱Sidecar（Go程序） | ~20MB |

### 11.3 Makefile

```makefile
# Makefile

# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

# 镜像仓库
REGISTRY ?= registry.xxx/agent-infra

# Go参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# 构建参数
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) \
         -X main.BuildTime=$(BUILD_TIME) \
         -X main.GitCommit=$(GIT_COMMIT)"

# ==================== 开发命令 ====================

.PHONY: all build run test lint clean

all: build

build: ## 构建控制面服务
	cd cmd/control-plane && $(GOBUILD) $(LDFLAGS) -o ../../bin/control-plane .

run: build ## 本地运行
	./bin/control-plane -config configs/local.yaml

test: ## 运行单元测试
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage: test ## 查看测试覆盖率
	$(GOCMD) tool cover -html=coverage.out

lint: ## 代码检查
	golangci-lint run ./...

clean: ## 清理构建产物
	$(GOCLEAN)
	rm -rf bin/
	rm -rf coverage.out

# ==================== 前端命令 ====================

.PHONY: web-install web-build web-dev

web-install: ## 安装前端依赖
	cd web/user-panel && npm install
	cd web/admin-panel && npm install

web-build: ## 构建前端
	cd web/user-panel && npm run build
	cd web/admin-panel && npm run build

web-dev: ## 前端开发模式
	cd web/user-panel && npm run dev

# ==================== Docker命令 ====================

.PHONY: docker-build-control-plane docker-build-frontend \
        docker-build-cli-runner docker-build-wrapper \
        docker-build-all docker-push

docker-build-control-plane: ## 构建控制面镜像
	docker build -f deploy/dockerfiles/Dockerfile.control-plane \
		-t $(REGISTRY)/control-plane:$(VERSION) \
		-t $(REGISTRY)/control-plane:latest .

docker-build-frontend: ## 构建前端镜像
	docker build -f deploy/dockerfiles/Dockerfile.frontend \
		-t $(REGISTRY)/frontend:$(VERSION) \
		-t $(REGISTRY)/frontend:latest .

docker-build-cli-runner: ## 构建CLI Runner镜像
	docker build -f deploy/dockerfiles/Dockerfile.cli-runner \
		-t $(REGISTRY)/cli-runner:$(VERSION) \
		-t $(REGISTRY)/cli-runner:latest .

docker-build-wrapper: ## 构建Agent Wrapper镜像
	docker build -f deploy/dockerfiles/Dockerfile.agent-wrapper \
		-t $(REGISTRY)/agent-wrapper:$(VERSION) \
		-t $(REGISTRY)/agent-wrapper:latest .

docker-build-all: docker-build-control-plane docker-build-frontend \
                  docker-build-cli-runner docker-build-wrapper
	## 构建所有镜像

docker-push: ## 推送镜像
	docker push $(REGISTRY)/control-plane:$(VERSION)
	docker push $(REGISTRY)/control-plane:latest
	docker push $(REGISTRY)/frontend:$(VERSION)
	docker push $(REGISTRY)/frontend:latest
	docker push $(REGISTRY)/cli-runner:$(VERSION)
	docker push $(REGISTRY)/cli-runner:latest
	docker push $(REGISTRY)/agent-wrapper:$(VERSION)
	docker push $(REGISTRY)/agent-wrapper:latest

# ==================== K8s命令 ====================

.PHONY: k8s-apply k8s-delete k8s-logs k8s-status

k8s-apply: ## 部署到K8s
	kubectl apply -f deploy/k8s/control-plane/
	kubectl apply -f deploy/k8s/sandbox/namespace.yaml

k8s-delete: ## 从K8s删除
	kubectl delete -f deploy/k8s/control-plane/

k8s-logs: ## 查看控制面日志
	kubectl logs -f -l app=control-plane -n control-plane

k8s-status: ## 查看部署状态
	kubectl get pods -n control-plane
	kubectl get pods -n sandbox

# ==================== 数据库命令 ====================

.PHONY: db-migrate db-rollback

db-migrate: ## 执行数据库迁移
	./scripts/migrate.sh up

db-rollback: ## 回滚数据库迁移
	./scripts/migrate.sh down

# ==================== 帮助 ====================

.PHONY: help

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
```

### 11.4 CI/CD流程

#### 11.4.1 CI流程（GitLab CI示例）

```yaml
# .gitlab-ci.yml

stages:
  - lint
  - test
  - build
  - deploy

variables:
  REGISTRY: registry.xxx/agent-infra
  VERSION: ${CI_COMMIT_SHORT_SHA}

lint:
  stage: lint
  image: golangci/golangci-lint:latest
  script:
    - golangci-lint run ./...

test:
  stage: test
  image: golang:1.22
  script:
    - go test -v -race -coverprofile=coverage.out ./...
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml

build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - make docker-build-all VERSION=${VERSION}
    - make docker-push VERSION=${VERSION}
  only:
    - main
    - tags

deploy-dev:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - kubectl set image deployment/control-plane
      control-plane=${REGISTRY}/control-plane:${VERSION}
      -n control-plane
    - kubectl set image deployment/frontend
      frontend=${REGISTRY}/frontend:${VERSION}
      -n control-plane
  only:
    - main
  when: manual

deploy-prod:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - kubectl set image deployment/control-plane
      control-plane=${REGISTRY}/control-plane:${VERSION}
      -n control-plane
  only:
    - tags
  when: manual
```

---

## 12. 测试策略

### 12.1 测试分层

```
┌─────────────────────────────────────────────────────────────────┐
│                        测试金字塔                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                         /‾‾‾‾‾‾\                                │
│                        /  E2E   \          5%                   │
│                       /──────────\                               │
│                      /  集成测试   \        20%                  │
│                     /──────────────\                             │
│                    /    单元测试     \      75%                  │
│                   /────────────────────\                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 12.2 单元测试

| 模块 | 测试内容 | 工具 |
|------|---------|------|
| Service层 | 业务逻辑验证 | Go testing + testify |
| Scheduler | 调度算法、限流逻辑 | Go testing |
| Executor | Pod管理逻辑 | Go testing + mock |
| 工具函数 | 边界条件、异常处理 | Go testing |

### 12.3 集成测试

| 场景 | 测试内容 | 工具 |
|------|---------|------|
| API端到端 | 完整请求流程 | Go testing + httptest |
| 数据库交互 | CRUD操作 | Go testing + test DB |
| Redis交互 | 队列操作 | Go testing + miniredis |

### 12.4 E2E测试（手动）

| 场景 | 测试步骤 |
|------|---------|
| 任务完整流程 | 创建任务 → 调度 → 执行 → 完成 |
| 人工干预 | 暂停任务 → 注入指令 → 恢复 → 完成 |
| 异常处理 | 模拟失败 → 重试 → 恢复 |

### 12.5 测试覆盖率目标

| 层级 | 目标覆盖率 |
|------|-----------|
| Service层 | > 80% |
| Scheduler | > 85% |
| Executor | > 75% |
| 总体 | > 75% |

---

## 13. 附录

### 13.1 参考资料

- [Claude Code CLI 文档](https://docs.anthropic.com/claude/docs/claude-code)
- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [Ant Design 组件库](https://ant.design/components/)

### 13.2 待确认事项

- [ ] OceanBase具体版本和连接参数
- [ ] 阿里云SLS Project和Logstore命名规范
- [ ] 京东行云Git API访问凭证配置方式
- [ ] 企业IAM对接的具体接口规范

---

*文档结束*
