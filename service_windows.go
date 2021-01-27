package service

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

var elog debug.Log

// Execute will be called at the start of the service,
// and the service will exit once Execute completes.
func (s *Service) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	elog.Info(1, fmt.Sprintf("Service %s started.", s.Name))
	go s.Exec()
loop:
	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			status <- c.CurrentStatus
			time.Sleep(100 * time.Millisecond)
			status <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			elog.Info(1, fmt.Sprintf("Stopping %s service(%d).", s.Name, c.Context))
			break loop
		default:
			elog.Error(1, fmt.Sprintf("Unexpected control request #%d", c))
		}
	}
	status <- svc.Status{State: svc.StopPending}
	return
}

// Install installs the service.
func (s *Service) Install() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.Name)
	if err == nil {
		service.Close()
		return fmt.Errorf("service %s already exists", s.Name)
	}
	if s.Desc == "" {
		s.Desc = s.Name
	}
	service, err = m.CreateService(s.Name, execPath, mgr.Config{
		StartType:   mgr.StartAutomatic,
		Description: s.Desc,
	})
	if err != nil {
		return err
	}
	defer service.Close()

	if err := eventlog.InstallAsEventCreate(s.Name, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		service.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}
	return nil
}

// Remove removes the service.
func (s *Service) Remove() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.Name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", s.Name)
	}
	defer service.Close()

	if err := service.Delete(); err != nil {
		return err
	}
	return eventlog.Remove(s.Name)
}

// Run runs the service.
func (s *Service) Run(isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(s.Name)
	} else {
		elog, err = eventlog.Open(s.Name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Starting %s service.", s.Name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	if err := run(s.Name, s); err != nil {
		elog.Error(1, fmt.Sprintf("Run %s service failed: %v", s.Name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped.", s.Name))
}

// Start starts the service.
func (s *Service) Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.Name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	return service.Start()
}

// Stop stops the service.
func (s *Service) Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	service, err := m.OpenService(s.Name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()
	status, err := service.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", svc.Stop, err)
	}
	timeout := time.Now().Add(10 * time.Second)
	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", svc.Stopped)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}

// Restart restarts the service.
func (s *Service) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// IsWindowsService reports whether the process is currently executing
// as a service.
func IsWindowsService() bool {
	is, _ := svc.IsWindowsService()
	return is
}
