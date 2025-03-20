package docker

// CreateModuleQueryExec creates a Evmos module query
func (m *Manager) CreateQueryExec(containerID, branch, moduleName, subCommand string, args ...string) (string, error) {
	cmd := []string{
		"gurud",
		"q",
		moduleName,
		subCommand,
	}
	cmd = append(cmd, args...)
	return m.CreateExec(cmd, containerID)
}
