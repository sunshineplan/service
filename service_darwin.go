package service

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>{{.Name}}</string>
    <key>WorkingDirectory</key>
    <string>{{.Dir}}</string>
    <key>ProgramArguments</key>
    <array>
      <string>{{.Path}}</string>{{range .Arguments}}
      <string>{{.}}</string>{{end}}
    </array>{{if .Environment}}
    <key>EnvironmentVariables</key>
    <dict>{{range $key, $value := .Environment}}
      <key>{{$key}}</key>
      <string>{{$value}}</string>{{end}}
    </dict>{{end}}
    <key>RunAtLoad</key>
    <true/>
  </dict>
</plist>
`

func (s *Service) plist() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.HomeDir + "/Library/LaunchAgents/" + strings.ToLower(s.Name) + ".plist", nil
}

func (s *Service) target() string {
	return fmt.Sprintf("gui/%d/%s", os.Getuid(), strings.ToLower(s.Name))
}

// Install installs the service.
func (s *Service) Install() error {
	plistPath, err := s.plist()
	if err != nil {
		return err
	}

	if _, err := os.Stat(plistPath); err == nil {
		return fmt.Errorf("plist %s exists", plistPath)
	}

	f, err := os.Create(plistPath)
	if err != nil {
		return err
	}
	defer f.Close()

	path, err := os.Executable()
	if err != nil {
		return err
	}

	format := struct {
		Name        string
		Dir         string
		Path        string
		Arguments   []string
		Environment map[string]string
	}{
		strings.ToLower(s.Name),
		filepath.Dir(path),
		path,
		s.Options.Arguments,
		s.Options.Environment,
	}

	if err := template.Must(template.New("").Parse(launchdPlist)).Execute(f, format); err != nil {
		return err
	}

	return run("launchctl", "bootstrap", "gui/"+strconv.Itoa(os.Getuid()), plistPath)
}

// Uninstall uninstalls the service.
func (s *Service) Uninstall() error {
	plistPath, err := s.plist()
	if err != nil {
		return err
	}

	if err := s.launchctl("bootout"); err != nil {
		return err
	}

	return os.Remove(plistPath)
}

// Start starts the service.
func (s *Service) Start() error {
	return s.launchctl("kickstart")
}

// Stop stops the service.
func (s *Service) Stop() error {
	return s.launchctl("kill", "SIGKILL")
}

// Restart restarts the service.
func (s *Service) Restart() error {
	return s.launchctl("kickstart", "-k")
}

// Status shows the service status.
func (s *Service) Status() error {
	return s.launchctl("print")
}

func (s *Service) launchctl(arg ...string) error {
	return run("launchctl", append(arg, s.target())...)
}
