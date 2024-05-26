package service

import (
	"log"
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
  log
  	Display log if present
` {
		t.Fatalf("wrong usage: %s", usage)
	}
	s.commands = nil
	s.m = make(map[string]command)
	var res string
	s.RegisterCommand("test", "test", func(arg ...string) error {
		res = strings.Join(arg, ",")
		return nil
	}, 2, true)
	s.RegisterCommand("hide", "test", func(arg ...string) error {
		res = strings.Join(arg, ",")
		return nil
	}, 2, false)
	if usage := s.Usage(); usage != `
service command:
  test
  	test
` {
		log.Print(s.m, s.commands)
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
