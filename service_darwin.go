package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
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

func (s *Service) getInfo() (string, string, error) {
	u, err := user.Current()
	if err != nil {
		return "", "", err
	}

	return u.Uid, u.HomeDir + "/Library/LaunchAgents/" + strings.ToLower(s.Name) + ".plist", nil
}

// Install installs the service.
func (s *Service) Install() error {
	uid, plistPath, err := s.getInfo()
	if err != nil {
		return err
	}

	if _, err := os.Stat(plistPath); err == nil {
		return fmt.Errorf("plist %s exists", plistPath)
	}

	f, err := os.OpenFile(plistPath, os.O_WRONLY|os.O_CREATE, 0644)
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

	return launchctl("bootstrap", "gui/"+uid, plistPath)
}

// Uninstall uninstalls the service.
func (s *Service) Uninstall() error {
	uid, plistPath, err := s.getInfo()
	if err != nil {
		return err
	}

	if err := launchctl("bootout", fmt.Sprintf("gui/%s/%s", uid, strings.ToLower(s.Name))); err != nil {
		return err
	}

	return os.Remove(plistPath)
}

// Start starts the service.
func (s *Service) Start() error {
	uid, _, err := s.getInfo()
	if err != nil {
		return err
	}

	return launchctl("kickstart", fmt.Sprintf("gui/%s/%s", uid, strings.ToLower(s.Name)))
}

// Stop stops the service.
func (s *Service) Stop() error {
	uid, _, err := s.getInfo()
	if err != nil {
		return err
	}

	return launchctl("kill", "SIGKILL", fmt.Sprintf("gui/%s/%s", uid, strings.ToLower(s.Name)))
}

// Restart restarts the service.
func (s *Service) Restart() error {
	uid, _, err := s.getInfo()
	if err != nil {
		return err
	}

	return launchctl("kickstart", "-k", fmt.Sprintf("gui/%s/%s", uid, strings.ToLower(s.Name)))
}

func launchctl(arg ...string) error {
	cmd := exec.Command("launchctl", arg...)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("execute %q failed: %v", cmd.String(), err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("run %q failed: %s", cmd.String(), exiterr.Stderr)
		}

		return fmt.Errorf("execute %q failed: %v", cmd.String(), err)
	}

	return nil
}
