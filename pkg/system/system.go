package system

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"summit/pkg/model"
	"syscall"

	"github.com/spf13/afero"
)

// InferSystemState infers the current system state by gathering information about installed packages,
// running services, existing users, and system configurations.
// It returns a SystemState struct containing this information or an error if any occurred.
func InferSystemState(runner CommandRunner, skipIntrinsicIgnores bool) (*model.SystemState, []model.IgnoredConfig, error) {
	packages, err := listInstalledPackages()
	if err != nil {
		return nil, nil, err
	}

	services, err := listServices()
	if err != nil {
		return nil, nil, err
	}

	users, err := listUsers(runner)
	if err != nil {
		return nil, nil, err
	}

	configs, ignored, err := listSystemConfigs(runner, skipIntrinsicIgnores)
	if err != nil {
		return nil, nil, err
	}

	return &model.SystemState{
		Packages: packages,
		Services: services,
		Users:    users,
		Configs:  configs,
	}, ignored, nil
}

// listInstalledPackages returns all installed packages in the system
func listInstalledPackages() ([]model.PackageState, error) {
	worldPath := "/etc/apk/world"
	content, err := afero.ReadFile(AppFs, worldPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s: %w", worldPath, err)
	}

	packageNames := strings.Split(string(content), "\n")
	packages := make([]model.PackageState, 0, len(packageNames))

	for _, packageName := range packageNames {
		packageName = strings.TrimSpace(packageName)
		if packageName != "" {
			packages = append(packages, model.PackageState{Name: packageName})
		}
	}

	return packages, nil
}

func listServices() ([]model.ServiceState, error) {
	servicesDir := "/etc/init.d"
	entries, err := afero.ReadDir(AppFs, servicesDir)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", servicesDir, err)
	}

	var services []model.ServiceState
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".sh") {
			continue
		}

		// Determine enabled status and runlevel
		var enabled bool
		var runlevel string

		// Check for symlinks in runlevels directories
		runlevels := []string{"boot", "default", "sysinit", "nonetwork", "shutdown"}
		for _, rl := range runlevels {
			runlevelPath := filepath.Join("/etc/runlevels", rl, name)
			_, err := AppFs.Stat(runlevelPath)
			if err == nil {
				enabled = true
				runlevel = rl
				break // Assuming a service is only enabled in one runlevel
			}
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("error checking runlevel path %s: %w", runlevelPath, err)
			}
		}

		// Include all services from /etc/init.d, even if not enabled
		// For disabled services, set runlevel to empty string
		services = append(services, model.ServiceState{
			Name:     name,
			Enabled:  enabled,
			Runlevel: runlevel,
		})
	}

	return services, nil
}

func listUsers(runner CommandRunner) ([]model.UserState, error) {
	// Build gid to group name map
	gidToName, err := buildGidToNameMap()
	if err != nil {
		return nil, err
	}

	passwdPath := "/etc/passwd"
	usersFile, err := AppFs.Open(passwdPath)
	if err != nil {
		return nil, fmt.Errorf("Error opening %s: %w", passwdPath, err)
	}
	defer usersFile.Close()

	users := []model.UserState{}
	scanner := bufio.NewScanner(usersFile)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}

		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		// filter users with UID < 1000
		if uid < 1000 {
			continue
		}

		// filter users with no shell
		if fields[6] == "" || strings.Contains(fields[6], "nologin") {
			continue
		}

		userName := fields[0]
		gid, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}
		primaryGroupName, ok := gidToName[gid]
		if !ok {
			primaryGroupName = fmt.Sprintf("%d", gid) // fallback to gid string
		}

		userGroups, err := listGroupsForUser(runner, userName)
		if err != nil {
			return nil, err
		}

		user := model.UserState{
			Name:         userName,
			Groups:       userGroups,
			PrimaryGroup: primaryGroupName,
		}
		users = append(users, user)
	}

	return users, nil
}

const groupFilePath = "/etc/group"

// buildGidToNameMap builds a map from gid to group name
func buildGidToNameMap() (map[int]string, error) {
	groupFile, err := AppFs.Open(groupFilePath)
	if err != nil {
		return nil, fmt.Errorf("Error opening %s: %w", groupFilePath, err)
	}
	defer groupFile.Close()

	gidToName := make(map[int]string)
	scanner := bufio.NewScanner(groupFile)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		gidToName[gid] = fields[0]
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", groupFilePath, err)
	}
	return gidToName, nil
}

// listSystemConfigs returns all system configs added or modified by the user
// sistem configs are configs stored in /etc folder
// Returns included configs and ignored configs with reasons
func listSystemConfigs(runner CommandRunner, skipIntrinsicIgnores bool) ([]model.SystemConfigState, []model.IgnoredConfig, error) {

	cmd := "apk audit"
	output, err := runner.Run("", cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("error running apk audit: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	configs := []model.SystemConfigState{}
	ignored := []model.IgnoredConfig{}
	modifiedFiles := []string{}

linesloop:
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 2 {
			continue
		}

		fileStatus := line[:1]
		filePath := strings.TrimSpace(line[1:])

		// Prepend / if not present
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}

		// Ignore files outside /etc
		if !strings.HasPrefix(filePath, "/etc") {
			continue
		}

		// ignore runlevel files
		if strings.HasPrefix(filePath, "/etc/runlevels") {
			if !skipIntrinsicIgnores {
				ignored = append(ignored, model.IgnoredConfig{Path: filePath, Reason: "intrinsic: runlevel files"})
			}
			continue
		}

		// ignore files that end with "-" or ".bak"
		if strings.HasSuffix(filePath, "-") || strings.HasSuffix(filePath, ".bak") {
			if !skipIntrinsicIgnores {
				ignored = append(ignored, model.IgnoredConfig{Path: filePath, Reason: "intrinsic: backup file"})
			}
			continue
		}

		// ignore predefined files and paths that are managed by the system or package manager
		// and should never be managed by Summit configurations
		ignoredPaths := []string{
			"/etc/passwd",    // User database, managed by system/user tools
			"/etc/group",     // Group database, managed by system/user tools
			"/etc/shadow",    // Shadow password file, managed by system/user tools
			"/etc/apk/world", // APK's list of installed packages, managed by apk
			"/etc/apk/keys",  // APK's trusted keys directory, managed by apk
			"/etc/apk/arch",  // System architecture for APK, set by Alpine installation
			"/etc/apk/protected_paths.d/ca-certificates.list", // APK's protected paths for ca-certificates
		}
		for _, ignoredPath := range ignoredPaths {
			if filePath == ignoredPath || strings.HasPrefix(filePath, ignoredPath) {
				if !skipIntrinsicIgnores {
					ignored = append(ignored, model.IgnoredConfig{Path: filePath, Reason: "intrinsic: " + ignoredPath})
				}
				continue linesloop
			}
		}

		if fileStatus != "X" {
			fileInfo, err := AppFs.Stat(filePath)
			if err != nil {
				return nil, nil, fmt.Errorf("error stating file %s: %w", filePath, err)
			}
			// We can't handle directories, we just skip them for now
			if fileInfo.IsDir() {
				if !skipIntrinsicIgnores {
					ignored = append(ignored, model.IgnoredConfig{Path: filePath, Reason: "intrinsic: directory"})
				}
				continue
			}
		}

		config := model.SystemConfigState{
			Path: filePath,
		}

		switch fileStatus {
		case "A": // File added
			config.Origin = model.OriginUserCreated
		case "U": // File updated
			config.Origin = model.OriginPackageModified
			modifiedFiles = append(modifiedFiles, filePath)
		case "X": // File deleted
			config.Deleted = true
		}

		configs = append(configs, config)
	}

	// Get package owner for modified files
	if len(modifiedFiles) > 0 {
		ownerMap, err := getPackageOwners(runner, modifiedFiles)
		if err != nil {
			return nil, nil, err
		}
		for i := range configs {
			if owner, ok := ownerMap[configs[i].Path]; ok {
				configs[i].OriginPackage = owner
			}
		}
	}

	// Read file content for added and updated files
	for i := range configs {
		if !configs[i].Deleted {
			content, err := afero.ReadFile(AppFs, configs[i].Path)
			if err != nil {
				return nil, nil, fmt.Errorf("error reading file %s: %w", configs[i].Path, err)
			}
			configs[i].Content = string(content)

			// Get FileInfo for mode and ownership
			fileInfo, err := AppFs.Stat(configs[i].Path)
			if err != nil {
				return nil, nil, fmt.Errorf("error stating file %s: %w", configs[i].Path, err)
			}

			// Get file mode, owner, and group
			if fileInfo.Sys() != nil {
				stat, ok := fileInfo.Sys().(*syscall.Stat_t)
				if !ok {
					return nil, nil, fmt.Errorf("error getting syscall.Stat_t for %s", configs[i].Path)
				} else {
					// Get owner
					uid := fmt.Sprint(stat.Uid)
					u, err := user.LookupId(uid)
					var ownerName string
					if err != nil {
						ownerName = uid // fallback to UID if lookup fails
					} else {
						ownerName = u.Username
					}

					// Get group
					gid := fmt.Sprint(stat.Gid)
					g, err := user.LookupGroupId(gid)
					var groupName string
					if err != nil {
						groupName = gid // fallback to GID if lookup fails
					} else {
						groupName = g.Name
					}

					configs[i].Owner = ownerName
					configs[i].Group = groupName
				}
			}

			configs[i].Mode = fmt.Sprintf("0%o", fileInfo.Mode().Perm())
		}
	}

	return configs, ignored, nil
}

func listGroupsForUser(runner CommandRunner, userName string) ([]string, error) {
	cmd := fmt.Sprintf("groups %s", userName)
	output, err := runner.Run("", cmd)
	if err != nil {
		// If the user doesn't exist, the command returns an error. Return an empty list in this case.
		if strings.Contains(err.Error(), "no such user") {
			return []string{}, nil
		}
		return []string{}, fmt.Errorf("error getting groups for user %s: %w", userName, err)
	}

	// The output of the groups command is a space-separated list of group names.
	groupsString := strings.TrimSpace(string(output))
	if groupsString == "" {
		return []string{}, nil
	}

	groups := strings.Split(groupsString, " ")

	return groups, nil
}

func getPackageOwners(runner CommandRunner, files []string) (map[string]string, error) {
	ownerMap := make(map[string]string)
	args := append([]string{"info", "--who-owns"}, files...)
	cmd := fmt.Sprintf("apk %s", strings.Join(args, " "))
	output, err := runner.Run("", cmd)
	if err != nil {
		// Ignore errors, as some files may not be owned by any package
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " is owned by ", 2)
		if len(parts) == 2 {
			path := strings.TrimSpace(parts[0])
			owner := strings.TrimSpace(parts[1])
			ownerMap[path] = owner
		}
	}

	return ownerMap, nil
}
