# Claude Code Instructions

@agent.md

## Preferences

- 用中文回复，代码和注释用英文
- commit 时自动加 Co-Authored-By: Claude <noreply@anthropic.com>
- 使用 superpowers 技能（writing-plans, subagent-driven-development 等）时，计划的任务结构必须对齐 agent.md §7 的完整工作流（Step 0-13），开发前创建 14 步 TaskCreate 列表

## Permissions

- 允许执行 make test, make lint, go test 命令
- 允许执行 gh pr create, gh issue 操作
