# kubectl-operator Claude Code Slash Commands

This directory contains slash commands that give Claude Code awareness of the kubectl-operator plugin.

## Available Commands

### `/operator`
**General help with kubectl operator plugin**

Use this for general questions about kubectl operator commands, workflows, and best practices.

**Example usage:**
```
/operator
How do I install the prometheus operator?
```

```
/operator
What's the difference between OLMv0 and OLMv1 commands?
```

### `/operator-install`
**Install a Kubernetes operator**

Focused helper for installing operators using `kubectl operator olmv1 install extension`.

**Example usage:**
```
/operator-install
I want to install the prometheus operator
```

```
/operator-install
Install cert-manager in the cert-manager namespace
```

### `/operator-catalog`
**Manage operator catalogs**

Helper for creating, listing, updating, and deleting ClusterCatalogs.

**Example usage:**
```
/operator-catalog
Create a catalog from quay.io/operatorhubio/catalog:latest
```

```
/operator-catalog
How do I check if my catalog is serving?
```

### `/operator-debug`
**Troubleshoot kubectl operator issues**

Debug helper that matches error messages to documented scenarios and provides fixes.

**Example usage:**
```
/operator-debug
I got error: "no bundles found for package prometheus-operator"
```

```
/operator-debug
My extension shows INSTALLED: False. How do I debug?
```

## How These Work

Each slash command is a Markdown file that:
1. Loads context about kubectl-operator
2. References `olmv1.md` for accurate, up-to-date documentation
3. Follows documented workflows and error scenarios
4. Provides step-by-step guidance with verification

## Knowledge Base

All commands reference `olmv1.md` which contains:
- Complete command syntax and flags
- Prerequisites for each operation
- Success verification steps
- Common error scenarios and fixes
- Recommended workflows
- Example outputs (YAML, JSON, table formats)

## Customization

You can edit any of these `.md` files to customize the behavior. The format is:

```markdown
---
description: Short description for the command list
---

Instructions for Claude Code...
```

## Adding New Commands

Create a new `.md` file in this directory with your command name:

```bash
.claude/commands/my-command.md
```

Then use it in Claude Code:
```
/my-command
```

## No MCP Server Required

These slash commands work without an MCP server - they're just Markdown files that expand into prompts. This makes them:
- Easy to version control
- Simple to edit
- Fast to load
- No dependencies to manage

## Documentation

For more on Claude Code slash commands, see:
https://docs.claude.com/en/docs/claude-code/custom-commands
