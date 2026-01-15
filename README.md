# Summit

![Summit](Summit.jpg)

Summit is a declarative, command-line tool for managing Alpine Linux systems. It allows you to define the desired state of your entire system—packages, services, users, and configuration files—in a single YAML file. Summit then intelligently figures out the necessary changes and applies them in a safe, transactional way.

## Core Philosophy

Inspired by modern infrastructure-as-code principles, Summit brings simplicity and predictability to system management. Stop writing imperative shell scripts and start declaring the state you want.

## Features

- Declarative configuration with YAML
- Automatic state detection and diffing
- Transactional applies with rollback on failure
- Dry-run mode for safe previews
- Intelligent file management (managed vs. unmanaged)
- Extensible action-based architecture

## Installation

Requires Go compiler.

```bash
git clone https://github.com/lfantonioli/summit
cd summit
go build .
sudo mv summit /usr/local/bin/
```

## Quick Start

1. Create `system.yaml`:

```yaml
packages:
  - name: htop

configs:
  - path: /etc/motd
    content: "Welcome to Summit!"
```

2. Check changes: `summit diff`

3. Dry run: `summit apply --dry-run`

4. Apply: `summit apply`

5. Dump current state: `summit dump > state.yaml`

## Commands

### Global Flags

- `--config <path>`: Config file path (default: `./system.yaml`)
- `--log-level <level>`: Log level (debug, info, warn, error)

### `summit apply`

Applies changes to match desired state.

**Flags:**
- `--dry-run`: Preview changes without applying
- `--prune-unmanaged`: Remove unmanaged files
- `--json`: JSON output (with --dry-run)

### `summit diff`

Shows differences between current and desired state.

**Flags:**
- `--prune-unmanaged`: Include unmanaged file deletions
- `--json`: JSON output

### `summit dump`

Outputs current system state in YAML.

**Flags:**
- `--json`: JSON output
- `--show-ignored`: List ignored files
- `--preview-ignores <config>`: Preview ignores from config
- `--raw`: Include security-sensitive files
- `--all-services`: Show all services

## Configuration

The `system.yaml` file defines desired state.

### Sections

- **packages**: List of packages to install via apk
- **services**: Services to enable/disable with runlevel
- **users**: System users (UID >= 1000) and groups
- **configs**: Files to manage with content, permissions, ownership
- **user-packages**: Per-user packages (pipx, npm)
- **ignored-configs**: Glob patterns for files to ignore
- **includes**: Compose configs from multiple files

### Example

```yaml
packages:
  - name: htop

services:
  - name: sshd
    enabled: true
    runlevel: default

users:
  - name: user
    groups: [wheel]

configs:
  - path: /etc/motd
    content: "Managed by Summit"
    mode: "0644"

ignored-configs:
  - /etc/ssh/ssh_host_*
```

## Development

- Run tests: `go test ./...`
- Unit tests use mocks for isolation
- Integration tests in `/test/integration`

## License

GPL-3.0-or-later
