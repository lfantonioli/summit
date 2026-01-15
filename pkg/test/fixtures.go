package test

import (
	"summit/pkg/model"
)

// SampleSystemState returns a basic SystemState for testing.
func SampleSystemState() *model.SystemState {
	return &model.SystemState{
		Packages: []model.PackageState{
			{Name: "htop"},
			{Name: "vim"},
		},
		Services: []model.ServiceState{
			{Name: "nginx", Enabled: true, Runlevel: "default"},
		},
		Users: []model.UserState{
			{Name: "testuser", Groups: []string{"wheel"}},
		},
		Configs: []model.SystemConfigState{
			{
				Path:    "/etc/motd",
				Content: "Welcome to Alpine Linux",
				Mode:    "0644",
				Owner:   "root",
				Group:   "root",
				Origin:  model.OriginUserCreated,
			},
		},
		UserPackages: []model.UserPackageState{
			{
				User: "testuser",
				Pipx: []string{"black", "ruff"},
				Npm:  []string{"typescript"},
			},
		},
	}
}

// SampleConfigYAML returns a sample YAML configuration string.
func SampleConfigYAML() string {
	return `packages:
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
  - path: /etc/motd
    content: |
      Welcome to Alpine Linux
    mode: "0644"
    owner: root
    group: root
user-packages:
  - user: testuser
    pipx:
      - black
      - ruff
    npm:
      - typescript
`
}

// InvalidConfigYAML returns YAML with syntax errors.
func InvalidConfigYAML() string {
	return `packages:
  - name: htop
    invalid_field: value
  - name: vim
services:
  - name: nginx
    enabled: true
    runlevel: default
    extra_field: invalid
users:
  - name: testuser
    groups:
      - wheel
configs:
  - path: /etc/motd
    content: "valid content"
    mode: "0644"
    owner: root
    group: root
    unknown_field: error
`
}

// LargeSystemState returns a SystemState with many entries for performance testing.
func LargeSystemState() *model.SystemState {
	state := &model.SystemState{}

	// Add 100 packages
	for i := 0; i < 100; i++ {
		state.Packages = append(state.Packages, model.PackageState{Name: "package" + string(rune(i))})
	}

	// Add 50 services
	for i := 0; i < 50; i++ {
		state.Services = append(state.Services, model.ServiceState{
			Name:     "service" + string(rune(i)),
			Enabled:  i%2 == 0,
			Runlevel: "default",
		})
	}

	// Add 20 users
	for i := 0; i < 20; i++ {
		state.Users = append(state.Users, model.UserState{
			Name:   "user" + string(rune(i)),
			Groups: []string{"wheel"},
		})
	}

	// Add 30 configs
	for i := 0; i < 30; i++ {
		state.Configs = append(state.Configs, model.SystemConfigState{
			Path:    "/etc/config" + string(rune(i)),
			Content: "content for config " + string(rune(i)),
			Mode:    "0644",
			Owner:   "root",
			Group:   "root",
			Origin:  model.OriginUserCreated,
		})
	}

	return state
}

// EmptySystemState returns an empty SystemState.
func EmptySystemState() *model.SystemState {
	return &model.SystemState{}
}
