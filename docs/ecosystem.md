# Ecosystem

`justz` exists because `just` is good. It does not replace `just`, and it works alongside the tools others have built around it.

| If you want to... | Good choices |
|---|---|
| Run tasks, write recipes, use modules and attributes | [`just`](https://github.com/casey/just) itself. Its manual is the reference for syntax. |
| Edit justfiles with syntax checks and completion in your editor | [`just-lsp`](https://github.com/terror/just-lsp), a language server for `just`. |
| Let an agent call your recipes through MCP | [`just-mcp`](https://github.com/PromptExecution/just-mcp), an MCP server for `just`. |
| Teach an agent `just` syntax and conventions | The skills [`cli-just`](https://github.com/paulrberg/dot-agents/tree/main/skills/cli-just), [`justfile-expert`](https://github.com/laurigates/claude-plugins/tree/main/tools-plugin/skills/justfile-expert) and [`justfile`](https://github.com/mizchi/skills/tree/main/justfile). |

## What `justz` adds

The tools above help you write and run justfiles. `justz` is about what happens once a project has many recipes: can a new person or an agent find the right command, and does the documentation still match the recipes? It brings a grouped command reference with examples, checks you can run in CI, one registry that the help and the completion share, global shortcuts that cannot hide a real command, and JSON output for tools.

## How they combine

Use `just-lsp` while you write, `justz check` before you commit, and `just-mcp` if your agent should run recipes itself. The syntax skills and the `justz` skills do not compete: install both, and your agent gets syntax help plus a convention for making the result discoverable.

Know a tool that belongs in this table? Open an issue. All of this builds on the work of the `just` maintainers.
