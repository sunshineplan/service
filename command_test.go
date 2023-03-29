package service

import (
	"strings"
	"testing"
)

func TestCommand(t *testing.T) {
	s := New()
	if usage := s.Usage(); usage != `
service command:
  install
  	Install service
  uninstall
  	Uninstall service
  remove
  	Remove service, equal uninstall
  run
  	Run service executor
  test
  	Run service test executor
  start
  	Start service
  stop
  	Stop service
  restart
  	Restart service
  status
  	Show service status info
  update
  	Update service files if update url is provided
` {
		t.Fatalf("wrong usage: %s", usage)
	}
	s.RegisterCommand("install", "", nil, 0)
	s.RegisterCommand("uninstall", "", nil, 0)
	s.RegisterCommand("remove", "", nil, 0)
	s.RegisterCommand("run", "", nil, 0)
	s.RegisterCommand("test", "", nil, 0)
	s.RegisterCommand("start", "", nil, 0)
	s.RegisterCommand("stop", "", nil, 0)
	s.RegisterCommand("restart", "", nil, 0)
	s.RegisterCommand("status", "", nil, 0)
	s.RegisterCommand("update", "", nil, 0)
	var res string
	s.RegisterCommand("test", "test", func(arg ...string) error {
		res = strings.Join(arg, ",")
		return nil
	}, 2)
	if usage := s.Usage(); usage != `
service command:
  test
  	test
` {
		t.Fatalf("wrong usage: %s", usage)
	}
	if err := s.ParseAndRun([]string{"test", "a", "b"}); err != nil {
		t.Error(err)
	} else if expect := "a,b"; res != expect {
		t.Errorf("expected %q, got %q", expect, res)
	}
	if err := s.ParseAndRun([]string{"start"}); err == nil {
		t.Error("expect error, got nil")
	}
	if err := s.ParseAndRun([]string{"test", "a"}); err == nil {
		t.Error("expect error, got nil")
	}
}
