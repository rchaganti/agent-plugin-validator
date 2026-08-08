# apv — Agent Plugin Validator

[![Continuous Integration](https://github.com/rchaganti/agent-plugin-validator/actions/workflows/ci.yml/badge.svg)](https://github.com/rchaganti/agent-plugin-validator/actions/workflows/ci.yml)
[![Release](https://github.com/rchaganti/agent-plugin-validator/actions/workflows/release.yml/badge.svg)](https://github.com/rchaganti/agent-plugin-validator/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rchaganti/agent-plugin-validator)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`apv` (**A**gent **P**lugin **V**alidator) is a lightweight, high-performance, **schema-driven** Go CLI tool for validating Agent Plugin manifests (`plugin.json`) against the open [Agent Plugins v1.0.0 specification](https://agent-plugins.org/specification).

---

## Key Features

- 🎯 **100% Schema-Driven Validation**: Zero hardcoded field assumptions in validator code. Utilizes JSON Schema Draft 2020-12 with full ECMAScript lookaround regex support (`(?!...`).
- ⚡ **Agent & CI/CD Ready**: 
  - `--format json` output for programmatic consumption by AI agents and pipeline steps.
  - `--quiet` / `-q` silent mode returning deterministic exit codes (`0` = valid, `1` = invalid, `2` = usage/runtime error).
  - TTY auto-detection and standard `NO_COLOR` environment variable support.
- 📦 **Embedded & Cached Schema Lifecycle**:
  - Embedded v1.0.0 default schema fallback (`plugin.schema.json`).
  - `apv schema update [url]` to fetch and cache updated schemas locally (`~/.apv/schemas/`).
  - `--schema <path|url>` for one-off custom schema validation.
- 🚀 **Shell Autocompletion**: Built-in tab completion for Bash, Zsh, Fish, and PowerShell (`apv completion <shell>`) with `.json` file completion for `validate` and `--schema`.
- ⚠️ **Spec-Compliant Warning Handling**:
  - Unrecognized top-level fields are classified as **Warnings** (`⚠`) and ignored per Spec §5.2 without failing validation.

---

## Installation

### Via `go install`
```bash
go install github.com/rchaganti/agent-plugin-validator@latest
```

### Download Pre-built Binaries
Download static binaries for Linux, macOS, and Windows (amd64/arm64) from the [Releases Page](https://github.com/rchaganti/agent-plugin-validator/releases).

---

## Quick Start

### 1. Validate a Manifest File
```bash
apv validate plugin.json
```

**Output**:
```text
✓ Using schema: Agent Plugins Manifest (embedded default)
✓ plugin.json is VALID
```

### 2. Machine-Readable Output for AI Agents & CI/CD
```bash
apv validate --format json plugin.json
```

**Output**:
```json
{
  "valid": true,
  "schema": {
    "source": "embedded default",
    "path": "(embedded)",
    "id": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
    "title": "Agent Plugins Manifest"
  },
  "errors": []
}
```

### 3. Read Manifest from Standard Input
```bash
cat plugin.json | apv validate -
```

### 4. Silent Bash Verification for Scripts
```bash
apv validate -q plugin.json || exit 1
```

---

## Schema Management Commands

| Command | Description |
|---|---|
| `apv schema show` | Displays active schema title, `$id`, source, and path |
| `apv schema update [url]` | Downloads and caches the schema from `https://agent-plugins.org/...` or custom URL |
| `apv schema reset` | Deletes the cached schema and reverts to the embedded default |

---

## Exit Codes

| Exit Code | Meaning |
|---|---|
| `0` | **Validation Succeeded**: Plugin manifest is valid (or has non-fatal warnings) |
| `1` | **Validation Failed**: Manifest contains fatal schema errors |
| `2` | **Usage / Runtime Error**: File not found, invalid JSON syntax, CLI flag error |

---

## Spec §5.2 Warnings vs Errors

| Violation Type | Severity | Exit Code | Result | Spec Rule |
|---|---|---|---|---|
| Missing `$schema` or `name` | **Fatal Error** | `1` | `valid: false` | Required fields |
| Invalid `name` pattern/length | **Fatal Error** | `1` | `valid: false` | Format constraints |
| Unknown top-level property | **Warning** | `0` | `valid: true` | Ignored per Spec §5.2 |

---

## Development & Testing

```bash
# Run unit tests
go test -v ./...

# Build local binary
go build -o apv .
```

---

## License

[MIT License](LICENSE) © 2026 Ravikanth Chaganti & Contributors.
