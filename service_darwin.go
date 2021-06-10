package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>{{html .Name}}</string>
    <key>ProgramArguments</key>
    <array>
      <string>{{html .Path}}</string>{{range .Arguments}}
      <string>{{html .}}</string>{{end}}
    </array>
    <key>RunAtLoad</key>
    <true/>
  </dict>
</plist>
`

func (s *Service) getPlistPath() (string, error) {
	return "~/Library/LaunchDaemons/" + strings.ToLower(s.Name) + ".plist", nil
}

// Install installs the service.
func (s *Service) Install() error {
	plistPath, err := s.getPlistPath()
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
		Name      string
		Path      string
		Arguments []string
	}{
		strings.ToLower(s.Name),
		path,
		s.Options.Arguments,
	}

	if err := template.Must(template.New("").Parse(launchdPlist)).Execute(f, format); err != nil {
		return err
	}

	return launchctl("bootstrap", "gui/$(id -u)", plistPath)
}

// Remove removes the service.
func (s *Service) Remove() error {
	if err := launchctl("bootout", "gui/$(id -u)/"+strings.ToLower(s.Name)); err != nil {
		return err
	}

	plistPath, err := s.getPlistPath()
	if err != nil {
		return err
	}

	return os.Remove(plistPath)
}

// Run runs the service.
func (s *Service) Run(isDebug bool) {
	s.Exec()
}

// Start starts the service.
func (s *Service) Start() error {
	return launchctl("kickstart", "gui/$(id -u)/"+strings.ToLower(s.Name))
}

// Stop stops the service.
func (s *Service) Stop() error {
	return launchctl("kill", "SIGKILL", "gui/$(id -u)/"+strings.ToLower(s.Name))
}

// Restart restarts the service.
func (s *Service) Restart() error {
	return launchctl("kickstart", "-k", "gui/$(id -u)/"+strings.ToLower(s.Name))
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

// IsWindowsService reports whether the process is currently executing
// as a service.
func IsWindowsService() bool {
	return false
}
