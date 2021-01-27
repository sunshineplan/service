package service

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
)

const systemdScript = `[Unit]
Description={{.Description}}
{{range .Dependencies}}{{println .}}{{end}}

[Service]
ExecStart={{.Path}} {{.Arguments}}
{{range .Others}}{{println .}}{{end}}

[Install]
WantedBy=multi-user.target
`

func (s *Service) unitFile() string {
	return "/etc/systemd/system/" + strings.ToLower(s.Name) + ".service"
}

// Install installs the service.
func (s *Service) Install() error {
	unitFile := s.unitFile()
	if _, err := os.Stat(unitFile); err == nil {
		return fmt.Errorf("Service %s exists", unitFile)
	}

	f, err := os.OpenFile(unitFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	path, err := os.Executable()
	if err != nil {
		return err
	}
	var format = &struct {
		Description  string
		Path         string
		Dependencies []string
		Arguments    string
		Others       []string
	}{
		s.Desc,
		path,
		s.Options.Dependencies,
		s.Options.Arguments,
		s.Options.Others,
	}

	if err := template.Must(template.New("").Parse(systemdScript)).Execute(f, format); err != nil {
		return err
	}
	return s.shell("enable")
}

// Remove removes the service.
func (s *Service) Remove() error {
	err := s.shell("disable")
	if err != nil {
		return err
	}
	return os.Remove(s.unitFile())
}

// Run runs the service.
func (s *Service) Run(isDebug bool) {
	s.Exec()
}

// Start starts the service.
func (s *Service) Start() error {
	return s.shell("start")
}

// Stop stops the service.
func (s *Service) Stop() error {
	return s.shell("stop")
}

// Restart restarts the service.
func (s *Service) Restart() error {
	return s.shell("restart")
}

func (s *Service) shell(action string) error {
	cmd := exec.Command("systemctl", action, strings.ToLower(s.Name))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Execute %q failed: %v", action, err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("Run %q failed: %s", action, exiterr.Stderr)
		}
		return fmt.Errorf("Execute %q failed: %v", action, err)
	}
	return nil
}

// IsWindowsService reports whether the process is currently executing
// as a service.
func IsWindowsService() bool {
	return false
}
