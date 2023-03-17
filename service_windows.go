package service

import (
	"os"

	"github.com/sunshineplan/utils/log"
	"golang.org/x/sys/windows/svc"
)

var _ svc.Handler = (*Service)(nil)

// Execute will be called at the start of the service,
// and the service will exit once Execute completes.
func (s *Service) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	defer func() { status <- svc.Status{State: svc.StopPending} }()

	go func() { s.done <- s.Exec() }()

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				var err error
				if s.Kill != nil {
					err = s.Kill()
				}
				s.done <- err
				return
			default:
			}
		case err := <-s.done:
			s.done <- err
			return
		}
	}
}

// Install installs the service.
func (s *Service) Install() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	if err := s.sc("create", "binPath=", execPath, "start=", "auto"); err != nil {
		return err
	}

	if s.Desc != "" {
		if err := s.sc("description", s.Desc); err != nil {
			s.Print(err)
		}
	}

	return nil
}

// Uninstall uninstalls the service.
func (s *Service) Uninstall() error {
	s.sc("stop")
	return s.sc("delete")
}

// Run runs the service.
func (s *Service) Run() error {
	if s.Exec == nil {
		return ErrNoExcute
	}
	if s.Logger == nil {
		s.Logger = log.Default()
	}

	s.done = make(chan error, 2)
	if err := svc.Run(s.Name, s); err != nil {
		return err
	}

	return <-s.done
}

// Start starts the service.
func (s *Service) Start() error {
	return s.sc("start")
}

// Stop stops the service.
func (s *Service) Stop() error {
	return s.sc("stop")
}

// Restart restarts the service.
func (s *Service) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// Status shows the service status.
func (s *Service) Status() error {
	return s.sc("queryex")
}

func (s *Service) sc(action string, arg ...string) error {
	return run("sc", append([]string{action, s.Name}, arg...))
}

// IsWindowsService reports whether the process is currently executing
// as a service.
func IsWindowsService() bool {
	is, _ := svc.IsWindowsService()
	return is
}
