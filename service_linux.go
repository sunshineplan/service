package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const systemdScript = `[Unit]
Description={{.Description}}
{{range .Dependencies}}{{println .}}{{end}}
[Service]
WorkingDirectory={{.Dir}}
ExecStart={{.Path}}{{range .Arguments}} {{.}}{{end}}{{if .Environment}}{{range $key, $value := .Environment}}
Environment={{$key}}={{$value}}{{end}}{{end}}
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
		Dir          string
		Path         string
		Dependencies []string
		Arguments    []string
		Environment  map[string]string
		Others       []string
	}{
		s.Desc,
		filepath.Dir(path),
		path,
		s.Options.Dependencies,
		s.Options.Arguments,
		s.Options.Environment,
		s.Options.Others,
	}

	if err := template.Must(template.New("").Parse(systemdScript)).Execute(f, format); err != nil {
		return err
	}

	return s.systemctl("enable")
}

// Uninstall uninstalls the service.
func (s *Service) Uninstall() error {
	s.systemctl("stop")

	if err := s.systemctl("disable"); err != nil {
		return err
	}

	return os.Remove(s.unitFile())
}

// Start starts the service.
func (s *Service) Start() error {
	return s.systemctl("start")
}

// Stop stops the service.
func (s *Service) Stop() error {
	return s.systemctl("stop")
}

// Restart restarts the service.
func (s *Service) Restart() error {
	return s.systemctl("restart")
}

// Status shows the service status.
func (s *Service) Status() error {
	return s.systemctl("status")
}

func (s *Service) reload() error {
	return nil
}

func (s *Service) systemctl(action string) error {
	return run("systemctl", action, strings.ToLower(s.Name))
}
