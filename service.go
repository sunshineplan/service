package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/sunshineplan/utils/log"
)

var ErrNoExcute = errors.New("service execute is not defined")

var defaultName = "Service"

// Service represents a windows service.
type Service struct {
	*log.Logger
	Name     string
	Desc     string
	Exec     func() error
	Kill     func() error
	TestExec func() error
	Options  Options

	commands []string
	m        map[string]command

	done chan error
}

// Options is Service options
type Options struct {
	Dependencies       []string
	Arguments          []string
	Environment        map[string]string
	Others             []string
	PIDFile            string
	UpdateURL          string
	RemoveBeforeUpdate []string
	ExcludeFiles       []string
}

// New creates a new service name.
func New() *Service {
	svc := &Service{Logger: log.Default(), Name: defaultName}
	svc.initCommand()
	return svc
}

func (s *Service) SetLogger(file, prefix string, flag int) *Service {
	s.Logger = log.New(file, prefix, flag)
	return s
}

// Test tests the service.
func (s *Service) Test() (err error) {
	if s.TestExec != nil {
		if err = s.TestExec(); err != nil {
			s.Println("Test failed:", err)
		} else {
			s.Print("Test pass.")
		}
	} else {
		s.Print("No test provided.")
	}
	return
}

// Remove is an alias for Uninstall.
func (s *Service) Remove() error {
	return s.Uninstall()
}

func run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("execute %q failed: %s", cmd.String(), err)
	}
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("run %q failed: %s", cmd.String(), exiterr.Stderr)
		}
		return fmt.Errorf("execute %q failed: %s", cmd.String(), err)
	}
	return nil
}
