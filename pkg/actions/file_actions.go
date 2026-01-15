package actions

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"summit/pkg/log"
	"summit/pkg/system"
	"syscall"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/afero"
)

// FileCreateAction creates a file.
type FileCreateAction struct {
	Path    string
	Content string
	Mode    string
	Owner   string
	Group   string
}

func (a *FileCreateAction) Description() string {
	return fmt.Sprintf("Create file %s", a.Path)
}

func (a *FileCreateAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Creating file", "path", a.Path, "owner", a.Owner, "group", a.Group, "mode", a.Mode)
	if err := afero.WriteFile(system.AppFs, a.Path, []byte(a.Content), 0644); err != nil {
		return err
	}
	if a.Mode != "" {
		mode, err := strconv.ParseUint(a.Mode, 8, 32)
		if err != nil {
			return err
		}
		if err := system.AppFs.Chmod(a.Path, os.FileMode(mode)); err != nil {
			return err
		}
	}
	if a.Owner != "" || a.Group != "" {
		var uid, gid int
		if a.Owner != "" {
			u, err := user.Lookup(a.Owner)
			if err != nil {
				return err
			}
			uid, _ = strconv.Atoi(u.Uid)
		} else {
			uid = -1 // Keep current owner
		}

		if a.Group != "" {
			g, err := user.LookupGroup(a.Group)
			if err != nil {
				return err
			}
			gid, _ = strconv.Atoi(g.Gid)
		} else {
			gid = -1 // Keep current group
		}

		if err := system.AppFs.Chown(a.Path, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

func (a *FileCreateAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back file creation", "path", a.Path)
	err := system.AppFs.Remove(a.Path)
	if err != nil {
		logger.Error("Failed to roll back file creation", "path", a.Path, "error", err)
	}
	return err
}

func (a *FileCreateAction) ExecutionDetails() []string {
	details := []string{fmt.Sprintf("create file: %s with permissions %s", a.Path, a.Mode)}
	if a.Owner != "" {
		details = append(details, fmt.Sprintf("set owner to %s", a.Owner))
	}
	if a.Group != "" {
		details = append(details, fmt.Sprintf("set group to %s", a.Group))
	}
	return details
}

// FileUpdateAction updates a file.
type FileUpdateAction struct {
	Path        string
	NewContent  string
	origContent string
	origMode    os.FileMode
}

func (a *FileUpdateAction) Description() string {
	return fmt.Sprintf("Update file %s", a.Path)
}

func (a *FileUpdateAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Updating file content", "path", a.Path)
	info, err := system.AppFs.Stat(a.Path)
	if err != nil {
		return err
	}
	a.origMode = info.Mode()
	content, err := afero.ReadFile(system.AppFs, a.Path)
	if err != nil {
		return err
	}
	a.origContent = string(content)
	return afero.WriteFile(system.AppFs, a.Path, []byte(a.NewContent), a.origMode)
}

func (a *FileUpdateAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back file update", "path", a.Path)
	err := afero.WriteFile(system.AppFs, a.Path, []byte(a.origContent), a.origMode)
	if err != nil {
		logger.Error("Failed to roll back file update", "path", a.Path, "error", err)
	}
	return err
}

func (a *FileUpdateAction) ExecutionDetails() []string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a.origContent, a.NewContent, false)
	return []string{
		fmt.Sprintf("update file: %s", a.Path),
		"--- diff ---",
		dmp.DiffPrettyText(diffs),
		"--- end diff ---",
	}
}

// FileDeleteAction deletes a file.
type FileDeleteAction struct {
	Path        string
	origContent string
	origMode    os.FileMode
	origOwner   string
	origGroup   string
}

func (a *FileDeleteAction) Description() string {
	return fmt.Sprintf("Delete file %s", a.Path)
}

func (a *FileDeleteAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Deleting file", "path", a.Path)
	info, err := system.AppFs.Stat(a.Path)
	if err != nil {
		return err
	}
	a.origMode = info.Mode()
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		u, err := user.LookupId(fmt.Sprint(stat.Uid))
		if err == nil {
			a.origOwner = u.Username
		}
		g, err := user.LookupGroupId(fmt.Sprint(stat.Gid))
		if err == nil {
			a.origGroup = g.Name
		}
	}

	content, err := afero.ReadFile(system.AppFs, a.Path)
	if err != nil {
		return err
	}
	a.origContent = string(content)
	return system.AppFs.Remove(a.Path)
}

func (a *FileDeleteAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back file deletion by restoring content", "path", a.Path)
	if err := afero.WriteFile(system.AppFs, a.Path, []byte(a.origContent), a.origMode); err != nil {
		logger.Error("Failed to restore file content during rollback", "path", a.Path, "error", err)
		return err
	}

	// Rollback ownership
	if a.origOwner != "" || a.origGroup != "" {
		logger.Info("Rolling back file ownership", "path", a.Path, "owner", a.origOwner, "group", a.origGroup)
		var uid, gid int
		if a.origOwner != "" {
			u, err := user.Lookup(a.origOwner)
			if err != nil {
				logger.Error("Failed to lookup original owner for rollback", "user", a.origOwner, "error", err)
				return err
			}
			uid, _ = strconv.Atoi(u.Uid)
		} else {
			uid = -1
		}
		if a.origGroup != "" {
			g, err := user.LookupGroup(a.origGroup)
			if err != nil {
				logger.Error("Failed to lookup original group for rollback", "group", a.origGroup, "error", err)
				return err
			}
			gid, _ = strconv.Atoi(g.Gid)
		} else {
			gid = -1
		}
		if err := system.AppFs.Chown(a.Path, uid, gid); err != nil {
			logger.Error("Failed to chown file during rollback", "path", a.Path, "error", err)
			return err
		}
	}
	return nil
}

func (a *FileDeleteAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("delete file: %s", a.Path)}
}

// FileRevertAction reverts a file to its package-provided state.
type FileRevertAction struct {
	Path            string
	OwnerPackage    string
	modifiedContent string
}

func (a *FileRevertAction) Description() string {
	return fmt.Sprintf("Revert file %s to state from package %s", a.Path, a.OwnerPackage)
}

func (a *FileRevertAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Reverting file to package version", "path", a.Path, "package", a.OwnerPackage)
	content, err := afero.ReadFile(system.AppFs, a.Path)
	if err != nil {
		return err
	}
	a.modifiedContent = string(content)

	// Get package version
	out, err := runner.Run("", fmt.Sprintf("apk info %s", a.OwnerPackage))
	if err != nil {
		return fmt.Errorf("could not get package info for %s: %w", a.OwnerPackage, err)
	}
	// The output of apk info is like: `musl-1.2.4_git20230717-r4 description:`
	// We want to extract the version, which is everything after the first hyphen.
	parts := strings.SplitN(string(out), "-", 2)
	if len(parts) < 2 {
		return fmt.Errorf("could not parse package version from: %s", string(out))
	}
	version := strings.Split(parts[1], " ")[0]

	// Construct path to cached apk
	cachedApkPath := fmt.Sprintf("/var/cache/apk/%s-%s.apk", a.OwnerPackage, version)

	// Check if the cached file exists
	if _, err := system.AppFs.Stat(cachedApkPath); err != nil {
		return fmt.Errorf("cached apk not found at %s: %w. You may need to run 'apk add --no-cache' to ensure packages are cached.", cachedApkPath, err)
	}

	logger.Info("Found cached apk", "path", cachedApkPath)

	// Create temp dir
	tempDir, err := afero.TempDir(system.AppFs, "", "summit-apk-")
	if err != nil {
		return fmt.Errorf("could not create temp dir: %w", err)
	}
	defer system.AppFs.RemoveAll(tempDir)

	// Extract file
	// The path in the archive is relative, but a.Path is absolute. We need to strip the leading "/"
	relPath := strings.TrimPrefix(a.Path, "/")
	_, err = runner.Run("", fmt.Sprintf("tar -xzf %s -C %s %s", cachedApkPath, tempDir, relPath))
	if err != nil {
		return fmt.Errorf("could not extract file from package: %w", err)
	}

	// Replace file
	extractedFilePath := fmt.Sprintf("%s/%s", tempDir, relPath)
	return system.AppFs.Rename(extractedFilePath, a.Path)
}

func (a *FileRevertAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back file revert", "path", a.Path)
	err := afero.WriteFile(system.AppFs, a.Path, []byte(a.modifiedContent), 0644)
	if err != nil {
		logger.Error("Failed to roll back file revert", "path", a.Path, "error", err)
	}
	return err
}

func (a *FileRevertAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: apk fix --reinstall %s", a.OwnerPackage)}
}

// FileChmodAction changes the mode of a file.
type FileChmodAction struct {
	Path     string
	Mode     string
	origMode os.FileMode
}

func (a *FileChmodAction) Description() string {
	return fmt.Sprintf("Chmod file %s to %s", a.Path, a.Mode)
}

func (a *FileChmodAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Changing file mode", "path", a.Path, "mode", a.Mode)
	info, err := system.AppFs.Stat(a.Path)
	if err != nil {
		return err
	}
	a.origMode = info.Mode()
	mode, err := strconv.ParseUint(a.Mode, 8, 32)
	if err != nil {
		return err
	}
	return system.AppFs.Chmod(a.Path, os.FileMode(mode))
}

func (a *FileChmodAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back file mode", "path", a.Path, "mode", a.origMode)
	err := system.AppFs.Chmod(a.Path, a.origMode)
	if err != nil {
		logger.Error("Failed to roll back file mode", "path", a.Path, "error", err)
	}
	return err
}

func (a *FileChmodAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("chmod file %s to %s", a.Path, a.Mode)}
}

// FileChownAction changes the owner of a file.
type FileChownAction struct {
	Path      string
	Owner     string
	Group     string
	origOwner string
	origGroup string
}

func (a *FileChownAction) Description() string {
	return fmt.Sprintf("Chown file %s to %s:%s", a.Path, a.Owner, a.Group)
}

func (a *FileChownAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Changing file ownership", "path", a.Path, "owner", a.Owner, "group", a.Group)
	// Get original owner and group
	info, err := system.AppFs.Stat(a.Path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("could not get syscall.Stat_t for %s", a.Path)
	}
	u, err := user.LookupId(fmt.Sprint(stat.Uid))
	if err == nil {
		a.origOwner = u.Username
	}
	g, err := user.LookupGroupId(fmt.Sprint(stat.Gid))
	if err == nil {
		a.origGroup = g.Name
	}

	var uid, gid int
	if a.Owner != "" {
		u, err := user.Lookup(a.Owner)
		if err != nil {
			return err
		}
		uid, _ = strconv.Atoi(u.Uid)
	} else {
		uid = -1 // Keep current owner
	}

	if a.Group != "" {
		g, err := user.LookupGroup(a.Group)
		if err != nil {
			return err
		}
		gid, _ = strconv.Atoi(g.Gid)
	} else {
		gid = -1 // Keep current group
	}

	return system.AppFs.Chown(a.Path, uid, gid)
}

func (a *FileChownAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back file ownership", "path", a.Path, "owner", a.origOwner, "group", a.origGroup)
	var uid, gid int
	if a.origOwner != "" {
		u, err := user.Lookup(a.origOwner)
		if err != nil {
			logger.Error("Failed to lookup original owner for rollback", "user", a.origOwner, "error", err)
			return err
		}
		uid, _ = strconv.Atoi(u.Uid)
	} else {
		uid = -1 // Keep current owner
	}

	if a.origGroup != "" {
		g, err := user.LookupGroup(a.origGroup)
		if err != nil {
			logger.Error("Failed to lookup original group for rollback", "group", a.origGroup, "error", err)
			return err
		}
		gid, _ = strconv.Atoi(g.Gid)
	} else {
		gid = -1 // Keep current group
	}

	err := system.AppFs.Chown(a.Path, uid, gid)
	if err != nil {
		logger.Error("Failed to chown file during rollback", "path", a.Path, "error", err)
	}
	return err
}

func (a *FileChownAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("chown file %s to %s:%s", a.Path, a.Owner, a.Group)}
}
