//go:build unix

package service

import (
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
)

// Run runs the service.
func (s *Service) Run() error {
	if s.Exec == nil {
		return ErrNoExcute
	}
	if pid := s.Options.PIDFile; pid != "" {
		if err := os.MkdirAll(filepath.Dir(pid), 0775); err != nil {
			s.Println("Failed to write pid file:", err)
		} else {
			if err := os.WriteFile(pid, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
				s.Println("Failed to write pid file:", err)
			} else {
				defer os.Remove(s.Options.PIDFile)
			}
		}
	}
	s.done = make(chan error, 2)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			s.Logger.Rotate()
		}
	}()
	if s.Kill != nil {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			for range c {
				s.done <- s.Kill()
				return
			}
		}()
	}
	go func() { s.done <- s.Exec() }()
	return <-s.done
}

// IsWindowsService reports whether the process is currently executing
// as a service.
func IsWindowsService() bool {
	return false
}
