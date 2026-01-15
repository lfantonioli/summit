package diff

import (
	"fmt"
	"testing"

	"summit/pkg/model"
	"summit/pkg/test"
)

// BenchmarkCalculatePlan_Small benchmarks diff calculation with small state sets
func BenchmarkCalculatePlan_Small(b *testing.B) {
	desired := &model.SystemState{
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
			{Path: "/etc/test.conf", Content: "test content"},
		},
	}

	current := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "vim"},
		},
		Services: []model.ServiceState{},
		Users:    []model.UserState{},
		Configs:  []model.SystemConfigState{},
	}

	runner := test.NewMockCommandRunner()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculatePlan(desired, current, runner, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalculatePlan_Medium benchmarks diff calculation with medium state sets
func BenchmarkCalculatePlan_Medium(b *testing.B) {
	desired := &model.SystemState{
		Packages: generatePackages(50),
		Services: generateServices(20),
		Users:    generateUsers(10),
		Configs:  generateConfigs(30),
	}

	current := &model.SystemState{
		Packages: generatePackages(25),
		Services: generateServices(10),
		Users:    generateUsers(5),
		Configs:  generateConfigs(15),
	}

	runner := test.NewMockCommandRunner()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculatePlan(desired, current, runner, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalculatePlan_Large benchmarks diff calculation with large state sets
func BenchmarkCalculatePlan_Large(b *testing.B) {
	desired := &model.SystemState{
		Packages: generatePackages(200),
		Services: generateServices(50),
		Users:    generateUsers(20),
		Configs:  generateConfigs(100),
		UserPackages: []model.UserPackageState{
			{User: "user1", Pipx: []string{"tool1", "tool2"}, Npm: []string{"pkg1", "pkg2"}},
			{User: "user2", Pipx: []string{"tool3"}, Npm: []string{"pkg3"}},
		},
	}

	current := &model.SystemState{
		Packages: generatePackages(150),
		Services: generateServices(30),
		Users:    generateUsers(15),
		Configs:  generateConfigs(80),
		UserPackages: []model.UserPackageState{
			{User: "user1", Pipx: []string{"tool1"}, Npm: []string{"pkg1"}},
		},
	}

	runner := test.NewMockCommandRunner()
	// Mock pipx and npm list commands
	for i := 0; i < 20; i++ {
		user := fmt.Sprintf("user%d", i+1)
		runner.SetResponse(user, "pipx list --json", []byte(`{"venvs":{}}`))
		runner.SetResponse(user, "npm list --json", []byte(`{"dependencies":{}}`))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculatePlan(desired, current, runner, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalculatePlan_WithUserPackages benchmarks diff calculation with user packages
func BenchmarkCalculatePlan_WithUserPackages(b *testing.B) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "pipx"},
			{Name: "npm"},
		},
		UserPackages: []model.UserPackageState{
			{
				User: "developer",
				Pipx: []string{"black", "isort", "flake8", "mypy", "pytest", "tox", "pre-commit", "poetry"},
				Npm:  []string{"typescript", "eslint", "prettier", "webpack", "nodemon", "pm2"},
			},
			{
				User: "admin",
				Pipx: []string{"black", "ruff", "pytest"},
				Npm:  []string{"typescript", "eslint"},
			},
		},
	}

	current := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "pipx"},
			{Name: "npm"},
		},
		UserPackages: []model.UserPackageState{},
	}

	runner := test.NewMockCommandRunner()
	// Mock pipx and npm list commands returning empty (no packages installed)
	runner.SetResponse("developer", "pipx list --json", []byte(`{"venvs":{}}`))
	runner.SetResponse("developer", "npm list --json", []byte(`{"dependencies":{}}`))
	runner.SetResponse("admin", "pipx list --json", []byte(`{"venvs":{}}`))
	runner.SetResponse("admin", "npm list --json", []byte(`{"dependencies":{}}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculatePlan(desired, current, runner, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalculatePlan_ConfigHeavy benchmarks diff calculation with many config files
func BenchmarkCalculatePlan_ConfigHeavy(b *testing.B) {
	desired := &model.SystemState{
		Configs: generateConfigs(500),
	}

	current := &model.SystemState{
		Configs: generateConfigs(400),
	}

	runner := test.NewMockCommandRunner()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculatePlan(desired, current, runner, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for generating test data

func generatePackages(count int) []model.PackageState {
	packages := make([]model.PackageState, count)
	for i := 0; i < count; i++ {
		packages[i] = model.PackageState{Name: fmt.Sprintf("package%d", i)}
	}
	return packages
}

func generateServices(count int) []model.ServiceState {
	services := make([]model.ServiceState, count)
	for i := 0; i < count; i++ {
		services[i] = model.ServiceState{
			Name:     fmt.Sprintf("service%d", i),
			Enabled:  i%2 == 0,
			Runlevel: "default",
		}
	}
	return services
}

func generateUsers(count int) []model.UserState {
	users := make([]model.UserState, count)
	for i := 0; i < count; i++ {
		users[i] = model.UserState{
			Name:   fmt.Sprintf("user%d", i),
			Groups: []string{"wheel", "docker"},
		}
	}
	return users
}

func generateConfigs(count int) []model.SystemConfigState {
	configs := make([]model.SystemConfigState, count)
	for i := 0; i < count; i++ {
		configs[i] = model.SystemConfigState{
			Path:    fmt.Sprintf("/etc/config%d.conf", i),
			Content: fmt.Sprintf("content for config %d", i),
			Mode:    "0644",
			Owner:   "root",
			Group:   "root",
		}
	}
	return configs
}
