package actions

import (
	"strings"
	"testing"

	"summit/pkg/model"
	"summit/pkg/system"
	"summit/pkg/test"

	"github.com/spf13/afero"
)

// BenchmarkFileCreateAction_Apply benchmarks file creation performance
func BenchmarkFileCreateAction_Apply(b *testing.B) {
	system.AppFs = afero.NewMemMapFs()
	runner := test.NewMockCommandRunner()
	logger := test.NewMockLogger(0)

	action := &FileCreateAction{
		Path:    "/test/file.txt",
		Content: "This is test content for benchmarking file creation performance.",
		Mode:    "0644",
		Owner:   "root",
		Group:   "root",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clean up previous file
		system.AppFs.Remove("/test/file.txt")

		err := action.Apply(runner, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFileCreateAction_Apply_LargeFile benchmarks creating large files
func BenchmarkFileCreateAction_Apply_LargeFile(b *testing.B) {
	system.AppFs = afero.NewMemMapFs()
	runner := test.NewMockCommandRunner()
	logger := test.NewMockLogger(0)

	// Create a large content string (~1MB)
	var largeContent strings.Builder
	baseContent := "This is a line of content that will be repeated many times to create a large file for benchmarking purposes. "
	for i := 0; i < 10000; i++ {
		largeContent.WriteString(baseContent)
	}

	action := &FileCreateAction{
		Path:    "/test/large-file.txt",
		Content: largeContent.String(),
		Mode:    "0644",
		Owner:   "root",
		Group:   "root",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clean up previous file
		system.AppFs.Remove("/test/large-file.txt")

		err := action.Apply(runner, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPackageInstallAction_Apply benchmarks package installation
func BenchmarkPackageInstallAction_Apply(b *testing.B) {
	runner := test.NewMockCommandRunner()
	logger := test.NewMockLogger(0)

	action := &PackageInstallAction{
		PackageName: "htop",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := action.Apply(runner, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUserPackageAction_Apply benchmarks user package actions
func BenchmarkUserPackageAction_Apply(b *testing.B) {
	runner := test.NewMockCommandRunner()
	logger := test.NewMockLogger(0)

	action := UserPackageAction{
		User:    "testuser",
		Manager: "pipx",
		Package: "black",
		State:   model.PackageStatePresent,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := action.Apply(runner, logger)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultipleActions benchmarks executing multiple actions in sequence
func BenchmarkMultipleActions(b *testing.B) {
	system.AppFs = afero.NewMemMapFs()
	runner := test.NewMockCommandRunner()
	logger := test.NewMockLogger(0)

	actions := []Action{
		&FileCreateAction{Path: "/etc/config1.conf", Content: "config1", Mode: "0644"},
		&FileCreateAction{Path: "/etc/config2.conf", Content: "config2", Mode: "0644"},
		&FileCreateAction{Path: "/etc/config3.conf", Content: "config3", Mode: "0644"},
		&PackageInstallAction{PackageName: "htop"},
		&PackageInstallAction{PackageName: "vim"},
		&UserPackageAction{User: "user1", Manager: "pipx", Package: "black", State: model.PackageStatePresent},
		&UserPackageAction{User: "user1", Manager: "npm", Package: "typescript", State: model.PackageStatePresent},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, action := range actions {
			// Clean up files for file actions
			if fileAction, ok := action.(*FileCreateAction); ok {
				system.AppFs.Remove(fileAction.Path)
			}

			err := action.Apply(runner, logger)
			if err != nil {
				b.Fatalf("Action %d failed: %v", j, err)
			}
		}
	}
}

// BenchmarkActionDescription benchmarks generating action descriptions
func BenchmarkActionDescription(b *testing.B) {
	actions := []Action{
		&FileCreateAction{Path: "/etc/test.conf", Content: "content", Mode: "0644"},
		&FileUpdateAction{Path: "/etc/test.conf", NewContent: "new content"},
		&FileDeleteAction{Path: "/etc/test.conf"},
		&PackageInstallAction{PackageName: "htop"},
		&PackageRemoveAction{PackageName: "htop"},
		&ServiceEnableAction{ServiceName: "nginx", Runlevel: "default"},
		&ServiceDisableAction{ServiceName: "nginx", Runlevel: "default"},
		&UserCreateAction{UserName: "testuser"},
		&UserRemoveAction{UserName: "testuser"},
		UserPackageAction{User: "user", Manager: "pipx", Package: "black", State: model.PackageStatePresent},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, action := range actions {
			_ = action.Description()
		}
	}
}

// BenchmarkActionExecutionDetails benchmarks generating execution details
func BenchmarkActionExecutionDetails(b *testing.B) {
	actions := []Action{
		&FileCreateAction{Path: "/etc/test.conf", Content: "content", Mode: "0644"},
		&FileUpdateAction{Path: "/etc/test.conf", NewContent: "new content"},
		&FileDeleteAction{Path: "/etc/test.conf"},
		&PackageInstallAction{PackageName: "htop"},
		&PackageRemoveAction{PackageName: "htop"},
		&ServiceEnableAction{ServiceName: "nginx", Runlevel: "default"},
		&ServiceDisableAction{ServiceName: "nginx", Runlevel: "default"},
		&UserCreateAction{UserName: "testuser"},
		&UserRemoveAction{UserName: "testuser"},
		UserPackageAction{User: "user", Manager: "pipx", Package: "black", State: model.PackageStatePresent},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, action := range actions {
			_ = action.ExecutionDetails()
		}
	}
}
