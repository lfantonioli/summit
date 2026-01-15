package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemState_Sort(t *testing.T) {
	state := &SystemState{
		Packages: []PackageState{
			{Name: "zebra"},
			{Name: "apple"},
		},
		Services: []ServiceState{
			{Name: "zservice"},
			{Name: "aservice"},
		},
		Users: []UserState{
			{Name: "zuser"},
			{Name: "auser"},
		},
		Configs: []SystemConfigState{
			{Path: "/z/path"},
			{Path: "/a/path"},
		},
		UserPackages: []UserPackageState{
			{User: "zuser"},
			{User: "auser"},
		},
	}

	state.Sort()

	assert.Equal(t, "apple", state.Packages[0].Name)
	assert.Equal(t, "zebra", state.Packages[1].Name)

	assert.Equal(t, "aservice", state.Services[0].Name)
	assert.Equal(t, "zservice", state.Services[1].Name)

	assert.Equal(t, "auser", state.Users[0].Name)
	assert.Equal(t, "zuser", state.Users[1].Name)

	assert.Equal(t, "/a/path", state.Configs[0].Path)
	assert.Equal(t, "/z/path", state.Configs[1].Path)

	assert.Equal(t, "auser", state.UserPackages[0].User)
	assert.Equal(t, "zuser", state.UserPackages[1].User)
}

func TestIsIntrinsicIgnore(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/etc/passwd", true},
		{"/etc/group", true},
		{"/etc/shadow", true},
		{"/etc/apk/world", true},
		{"/etc/apk/keys/somekey", true},
		{"/etc/runlevels/default/sshd", true},
		{"/etc/hosts", false},
		{"/etc/ssh/sshd_config", false},
		{"/etc/file-", true},
		{"/etc/file.bak", true},
		{"/etc/normal", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, isIntrinsicIgnore(tt.path))
		})
	}
}
