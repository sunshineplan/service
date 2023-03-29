package service

import (
	"fmt"
	"strings"
)

type command struct {
	fn    func(arg ...string) error
	args  int
	usage string
}

func (s *Service) initCommand() {
	s.m = make(map[string]command)
	s.RegisterCommand("install", "Install service", wrapFunc(s.Install), 0)
	s.RegisterCommand("uninstall", "Uninstall service", wrapFunc(s.Uninstall), 0)
	s.RegisterCommand("remove", "Remove service, equal uninstall", wrapFunc(s.Remove), 0)
	s.RegisterCommand("run", "Run service executor", wrapFunc(s.Run), 0)
	s.RegisterCommand("test", "Run service test executor", wrapFunc(s.Test), 0)
	s.RegisterCommand("start", "Start service", wrapFunc(s.Start), 0)
	s.RegisterCommand("stop", "Stop service", wrapFunc(s.Stop), 0)
	s.RegisterCommand("restart", "Restart service", wrapFunc(s.Restart), 0)
	s.RegisterCommand("status", "Show service status info", wrapFunc(s.Status), 0)
	s.RegisterCommand("update", "Update service files if update url is provided", wrapFunc(s.Update), 0)
}

func (s *Service) RegisterCommand(name, usage string, fn func(arg ...string) error, args int) {
	name = strings.ToLower(name)
	if _, ok := s.m[name]; !ok {
		s.commands = append(s.commands, name)
	}
	s.m[name] = command{fn, args, usage}
}

func (s *Service) ParseAndRun(args []string) error {
	if IsWindowsService() {
		return s.Run()
	}

	switch len(args) {
	case 0:
		return s.Run()
	default:
		if cmd, ok := s.m[strings.ToLower(args[0])]; ok && cmd.fn != nil {
			if a := args[1:]; len(a) == cmd.args {
				return cmd.fn(a...)
			} else {
				return fmt.Errorf("%s need %d arguments", args[0], cmd.args)
			}
		}
		return fmt.Errorf("unknown arguments: %s", strings.Join(args, " "))
	}
}

func (s *Service) Usage() string {
	var b strings.Builder
	b.WriteString("\nservice command:\n")
	for _, i := range s.commands {
		if s.m[i].fn != nil {
			fmt.Fprint(&b, "  ", i, "\n")
			fmt.Fprint(&b, "  \t", s.m[i].usage, "\n")
		}
	}
	return b.String()
}

func wrapFunc(fn func() error) func(arg ...string) error {
	return func(arg ...string) error { return fn() }
}
