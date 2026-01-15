package diff

import (
	"summit/pkg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDependencies_Success(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "pipx"},
			{Name: "npm"},
		},
		Services: []model.ServiceState{
			{Name: "sshd", Enabled: true},
		},
		UserPackages: []model.UserPackageState{
			{
				User: "mino",
				Pipx: []string{"black"},
				Npm:  []string{"prettier"},
			},
		},
	}

	current := &model.SystemState{
		Services: []model.ServiceState{
			{Name: "sshd"},
			{Name: "crond"},
		},
		Users: []model.UserState{
			{Name: "mino"},
		},
	}

	err := ValidateDependencies(desired, current)
	assert.NoError(t, err)
}

func TestValidateDependencies_MissingUserPackageDeps(t *testing.T) {
	desired := &model.SystemState{
		UserPackages: []model.UserPackageState{
			{
				User: "mino",
				Pipx: []string{"black"},
			},
		},
	}

	current := &model.SystemState{
		Users: []model.UserState{
			{Name: "mino"},
		},
	}

	err := ValidateDependencies(desired, current)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user packages require 'pipx' to be installed")
}

func TestValidateDependencies_MissingService(t *testing.T) {
	desired := &model.SystemState{
		Services: []model.ServiceState{
			{Name: "non-existent-service", Enabled: true},
		},
	}

	current := &model.SystemState{}

	err := ValidateDependencies(desired, current)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service 'non-existent-service' not found")
}

func TestValidateDependencies_MissingUser(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "pipx"},
		},
		UserPackages: []model.UserPackageState{
			{
				User: "non-existent-user",
				Pipx: []string{"black"},
			},
		},
	}

	current := &model.SystemState{}

	err := ValidateDependencies(desired, current)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user 'non-existent-user' not found for user-packages")
}

func TestValidateDependencies_MultipleErrors(t *testing.T) {
	desired := &model.SystemState{
		Services: []model.ServiceState{
			{Name: "non-existent-service", Enabled: true},
		},
		UserPackages: []model.UserPackageState{
			{
				User: "non-existent-user",
				Pipx: []string{"black"},
			},
		},
	}

	current := &model.SystemState{}

	err := ValidateDependencies(desired, current)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user packages require 'pipx' to be installed")
	assert.Contains(t, err.Error(), "service 'non-existent-service' not found")
	assert.Contains(t, err.Error(), "user 'non-existent-user' not found for user-packages")
}
