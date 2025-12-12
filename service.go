package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"

	"github.com/sunshineplan/utils/log"
)

// ErrNoExcute is the error returned when the service execute is not specified.
var ErrNoExcute = errors.New("service execute is not specified")

var defaultName = "Service"

// Service represents a service.
type Service struct {
	*log.Logger
	Name      string
	Desc      string
	Exec      func() error
	Kill      func() error
	TestExec  func() error
	Options   Options
	DebugAddr string

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

// SetLogger sets service's log file.
func (s *Service) SetLogger(file, prefix string, flag int) *Service {
	s.Logger = log.New(file, prefix, flag)
	return s
}

// SetDebug sets debug address for pprof.
func (s *Service) SetDebug(addr string) *Service {
	s.DebugAddr = addr
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

// Run runs the service.
func (s *Service) Run() error {
	if s.DebugAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		var server *http.Server
		go func() {
			server = &http.Server{Addr: s.DebugAddr, Handler: mux}
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				s.Println("Failed to start debug server:", err)
			}
		}()
		defer server.Shutdown(context.Background())
	}
	return s.run()
}

// Remove is an alias for Uninstall.
func (s *Service) Remove() error {
	return s.Uninstall()
}

func runCommand(name string, arg ...string) error {
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
