package service

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
	"time"
)

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>{{html .Name}}</string>
    <key>ProgramArguments</key>
    <array>
      <string>{{html .Path}}</string>
      <string>{{html .Arguments}}</string>
    </array>
    <key>KeepAlive</key>
    <true/>
  </dict>
</plist>
`

func (s *Service) getServiceFilePath() (string, error) {
	return "/Library/LaunchDaemons/" + s.Name + ".plist", nil
}

// Install installs the service.
func (s *Service) Install() error {
	confPath, err := s.getServiceFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(confPath); err == nil {
		return fmt.Errorf("Service %s exists", confPath)
	}

	f, err := os.OpenFile(confPath, os.O_WRONLY|os.O_CREATE, 0644)
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
		Arguments string
	}{
		strings.ToLower(s.Name),
		path,
		s.Options.Arguments,
	}

	return template.Must(template.New("").Parse(launchdPlist)).Execute(f, format)
}

// Remove removes the service.
func (s *Service) Remove() error {
	s.Stop()

	confPath, err := s.getServiceFilePath()
	if err != nil {
		return err
	}

	return os.Remove(confPath)
}

// Run runs the service.
func (s *Service) Run(isDebug bool) {
	s.Exec()
}

// Start starts the service.
func (s *Service) Start() error {
	return s.cmd("load")
}

// Stop stops the service.
func (s *Service) Stop() error {
	return s.cmd("unload")
}

// Restart restarts the service.
func (s *Service) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}

	time.Sleep(time.Second)

	return s.Start()
}

func (s *Service) cmd(action string) error {
	confPath, err := s.getServiceFilePath()
	if err != nil {
		return err
	}

	cmd := exec.Command("launchctl", action, confPath)

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
