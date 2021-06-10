package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

const systemdScript = `[Unit]
Description={{.Description}}
{{range .Dependencies}}{{println .}}{{end}}
[Service]
ExecStart={{.Path}}{{range .Arguments}} {{.}}{{end}}
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

	format := struct {
		Description  string
		Path         string
		Dependencies []string
		Arguments    []string
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

	return s.cmd("enable")
}

// Remove removes the service.
func (s *Service) Remove() error {
	s.cmd("stop")

	if err := s.cmd("disable"); err != nil {
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
	return s.cmd("start")
}

// Stop stops the service.
func (s *Service) Stop() error {
	return s.cmd("stop")
}

// Restart restarts the service.
func (s *Service) Restart() error {
	return s.cmd("restart")
}

func (s *Service) cmd(action string) error {
	cmd := exec.Command("systemctl", action, strings.ToLower(s.Name))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("execute %q failed: %v", action, err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("run %q failed: %s", action, exiterr.Stderr)
		}

		return fmt.Errorf("execute %q failed: %v", action, err)
	}

	return nil
}

// IsWindowsService reports whether the process is currently executing
// as a service.
func IsWindowsService() bool {
	return false
}
