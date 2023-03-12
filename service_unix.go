//go:build unix

package service

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/sunshineplan/utils/log"
)

// Run runs the service.
func (s *Service) Run() error {
	if s.Exec == nil {
		return ErrNoExcute
	}
	if s.Logger == nil {
		s.Logger = log.Default()
	}
	if s.Options.PIDFile != "" {
		if err := os.WriteFile(s.Options.PIDFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			s.Println("Failed to write pid file:", err)
		} else {
			defer os.Remove(s.Options.PIDFile)
		}
	}
	s.done = make(chan error, 1)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGHUP:
				s.Logger.Rotate()
			case syscall.SIGINT, syscall.SIGTERM:
				if s.Kill != nil {
					if err := s.Kill(); err != nil {
						s.Print(err)
					}
				} else {
					close(s.done)
				}
				return
			}
		}
	}()
	go func() {
		s.done <- s.Exec()
		close(s.done)
	}()
	return <-s.done
}

// Debug debugs the service.
func (s *Service) Debug() error {
	return s.Run()
}

// IsWindowsService reports whether the process is currently executing
// as a service.
func IsWindowsService() bool {
	return false
}
