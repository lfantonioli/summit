package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"summit/pkg/test"
	"testing"
)

// BenchmarkLoadConfig_Small benchmarks loading a small configuration
func BenchmarkLoadConfig_Small(b *testing.B) {
	logger := test.NewMockLogger(slog.LevelError)

	// Create temp file
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	configYAML := `packages:
  - name: htop
  - name: vim
services:
  - name: nginx
    enabled: true
    runlevel: default
users:
  - name: testuser
    groups:
      - wheel
configs:
  - path: /etc/test.conf
    content: "test content"
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configPath, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadConfig_Medium benchmarks loading a medium-sized configuration
func BenchmarkLoadConfig_Medium(b *testing.B) {
	logger := test.NewMockLogger(slog.LevelError)

	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Generate medium config with 50 packages, 20 services, 10 users, 30 configs
	var configYAML string
	configYAML += "packages:\n"
	for i := 0; i < 50; i++ {
		configYAML += fmt.Sprintf("  - name: package%d\n", i)
	}
	configYAML += "services:\n"
	for i := 0; i < 20; i++ {
		configYAML += fmt.Sprintf("  - name: service%d\n    enabled: true\n    runlevel: default\n", i)
	}
	configYAML += "users:\n"
	for i := 0; i < 10; i++ {
		configYAML += fmt.Sprintf("  - name: user%d\n    groups:\n      - wheel\n", i)
	}
	configYAML += "configs:\n"
	for i := 0; i < 30; i++ {
		configYAML += fmt.Sprintf("  - path: /etc/config%d.conf\n    content: \"content for config %d\"\n", i, i)
	}

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configPath, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadConfig_Large benchmarks loading a large configuration
func BenchmarkLoadConfig_Large(b *testing.B) {
	logger := test.NewMockLogger(slog.LevelError)

	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Generate large config with 200 packages, 50 services, 20 users, 100 configs
	var configYAML string
	configYAML += "packages:\n"
	for i := 0; i < 200; i++ {
		configYAML += fmt.Sprintf("  - name: package%d\n", i)
	}
	configYAML += "services:\n"
	for i := 0; i < 50; i++ {
		configYAML += fmt.Sprintf("  - name: service%d\n    enabled: true\n    runlevel: default\n", i)
	}
	configYAML += "users:\n"
	for i := 0; i < 20; i++ {
		configYAML += fmt.Sprintf("  - name: user%d\n    groups:\n      - wheel\n      - docker\n", i)
	}
	configYAML += "configs:\n"
	for i := 0; i < 100; i++ {
		configYAML += fmt.Sprintf("  - path: /etc/config%d.conf\n    content: \"This is a longer content string for config %d to test parsing performance with larger content blocks.\"\n    mode: \"0644\"\n    owner: root\n    group: root\n", i, i)
	}
	configYAML += "user-packages:\n"
	for i := 0; i < 10; i++ {
		configYAML += fmt.Sprintf("  - user: user%d\n    pipx:\n      - tool%d\n    npm:\n      - package%d\n", i, i, i)
	}

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configPath, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadConfig_Complex benchmarks loading a configuration with complex nested structures
func BenchmarkLoadConfig_Complex(b *testing.B) {
	logger := test.NewMockLogger(slog.LevelError)

	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configYAML := `packages:
  - name: htop
  - name: vim
  - name: git
services:
  - name: nginx
    enabled: true
    runlevel: default
  - name: sshd
    enabled: false
    runlevel: boot
users:
  - name: admin
    groups:
      - wheel
      - sudo
      - docker
      - kubernetes
  - name: developer
    groups:
      - wheel
      - docker
configs:
  - path: /etc/nginx/nginx.conf
    content: |
      user nginx;
      worker_processes auto;
      error_log /var/log/nginx/error.log;
      pid /run/nginx.pid;

      events {
          worker_connections 1024;
      }

      http {
          include /etc/nginx/mime.types;
          default_type application/octet-stream;

          server {
              listen 80;
              server_name localhost;
              location / {
                  root /usr/share/nginx/html;
                  index index.html index.htm;
              }
          }
      }
    mode: "0644"
    owner: root
    group: root
  - path: /etc/ssh/sshd_config
    content: |
      Port 22
      AddressFamily any
      ListenAddress 0.0.0.0
      ListenAddress ::

      HostKey /etc/ssh/ssh_host_rsa_key
      HostKey /etc/ssh/ssh_host_ecdsa_key
      HostKey /etc/ssh/ssh_host_ed25519_key

      SyslogFacility AUTH
      LogLevel INFO

      LoginGraceTime 2m
      PermitRootLogin prohibit-password
      StrictModes yes
      MaxAuthTries 6
      MaxSessions 10

      PubkeyAuthentication yes
      AuthorizedKeysFile .ssh/authorized_keys
      PasswordAuthentication no
      PermitEmptyPasswords no

      ChallengeResponseAuthentication no
      UsePAM yes
      PrintMotd no
      PrintLastLog yes
      TCPKeepAlive yes
      ClientAliveInterval 30
      ClientAliveCountMax 3

      UseDNS no
      PidFile /run/sshd.pid
      MaxStartups 10:30:100
      PermitTunnel no
      ChrootDirectory none
      VersionAddendum none

      Banner none
      AllowTcpForwarding yes
      AllowAgentForwarding yes
      AllowStreamLocalForwarding yes
      GatewayPorts no
      X11Forwarding no
      X11DisplayOffset 10
      X11UseLocalhost yes
      PermitTTY yes
      PrintMotd no
      PrintLastLog yes
      TCPKeepAlive yes
      PermitUserEnvironment no
      Compression delayed
      ClientAliveInterval 0
      ClientAliveCountMax 3
      UseDNS no
      PidFile /run/sshd.pid
      MaxStartups 10:30:100
      PermitTunnel no
      ChrootDirectory none
      VersionAddendum none
    mode: "0600"
    owner: root
    group: root
user-packages:
  - user: admin
    pipx:
      - black
      - isort
      - flake8
      - mypy
      - pytest
      - tox
      - pre-commit
      - poetry
    npm:
      - typescript
      - eslint
      - prettier
      - webpack
      - nodemon
      - pm2
  - user: developer
    pipx:
      - black
      - ruff
      - pytest
    npm:
      - typescript
      - eslint
ignored-configs:
  - "*.bak"
  - "*.tmp"
  - "/etc/hostname"
  - "/etc/resolv.conf"
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configPath, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}
