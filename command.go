package service

import (
	"fmt"
	"os"
	"strings"
)

type command struct {
	fn      func(arg ...string) error
	args    int
	usage   string
	display bool
}

func (s *Service) initCommand() {
	s.m = make(map[string]command)
	s.RegisterCommand("install", "Install service", wrapFunc(s.Install), 0, true)
	s.RegisterCommand("uninstall", "Uninstall service", wrapFunc(s.Uninstall), 0, true)
	s.RegisterCommand("remove", "Remove service, equal uninstall", wrapFunc(s.Remove), 0, true)
	s.RegisterCommand("run", "Run service executor", wrapFunc(s.Run), 0, true)
	s.RegisterCommand("test", "Run service test executor", wrapFunc(s.Test), 0, true)
	s.RegisterCommand("start", "Start service", wrapFunc(s.Start), 0, true)
	s.RegisterCommand("stop", "Stop service", wrapFunc(s.Stop), 0, true)
	s.RegisterCommand("restart", "Restart service", wrapFunc(s.Restart), 0, true)
	s.RegisterCommand("status", "Show service status info", wrapFunc(s.Status), 0, true)
	s.RegisterCommand("update", "Update service files if update url is provided", wrapFunc(s.Update), 0, true)
	s.RegisterCommand("log", "Display log if present", func(arg ...string) error {
		if s.Logger == nil {
			fmt.Println("no log file is set")
			return nil
		}
		file := s.File()
		fmt.Println("log file:", file)
		if strings.HasPrefix(file, "/dev/") {
			return nil
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(b)
		return err
	}, 0, true)
}

// RegisterCommand registers command according to name, and it will check the number of arguments
// if args is not a negative number.
func (s *Service) RegisterCommand(name, usage string, fn func(arg ...string) error, args int, display bool) {
	name = strings.ToLower(name)
	if _, ok := s.m[name]; !ok {
		s.commands = append(s.commands, name)
	}
	s.m[name] = command{fn, args, usage, display}
}

// ParseAndRun parses args and runs the service.
func (s *Service) ParseAndRun(args []string) error {
	if IsWindowsService() {
		return s.Run()
	}

	switch len(args) {
	case 0:
		return s.Run()
	default:
		if s.Logger.Writer() != os.Stderr {
			s.SetExtra(os.Stderr)
		}
		if cmd, ok := s.m[strings.ToLower(args[0])]; ok && cmd.fn != nil {
			if a := args[1:]; cmd.args < 0 || len(a) == cmd.args {
				return cmd.fn(a...)
			} else {
				return fmt.Errorf("%s need %d arguments", args[0], cmd.args)
			}
		}
		return fmt.Errorf("unknown arguments: %s", strings.Join(args, " "))
	}
}

// Usage returns service usage.
func (s *Service) Usage() string {
	var b strings.Builder
	b.WriteString("\nservice command:\n")
	for _, i := range s.commands {
		if command := s.m[i]; command.display && command.fn != nil {
			fmt.Fprint(&b, "  ", i, "\n")
			fmt.Fprint(&b, "  \t", s.m[i].usage, "\n")
		}
	}
	return b.String()
}

func wrapFunc(fn func() error) func(arg ...string) error {
	return func(arg ...string) error { return fn() }
}
