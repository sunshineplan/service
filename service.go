package service

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sunshineplan/utils/archive"
	"github.com/sunshineplan/utils/log"
	"github.com/sunshineplan/utils/progressbar"
)

const Usage = `
service command:
  install
    	Install service
  uninstall/remove
    	Uninstall service
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
    	Show service status
  update
    	Update service files if update url is provided
`

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
	return &Service{Logger: log.Default(), Name: defaultName}
}

func (s *Service) SetLogger(file, prefix string, flag int) *Service {
	s.Logger = log.New(file, prefix, flag)
	return s
}

// Update updates the service's installed files.
func (s *Service) Update() error {
	if s.Options.UpdateURL == "" {
		return fmt.Errorf("no update url provided")
	}
	if s.Logger == nil {
		s.Logger = log.Default()
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}

	resp, err := http.Get(s.Options.UpdateURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	total, err := strconv.Atoi(resp.Header.Get("content-length"))
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	pb := progressbar.New(total).SetUnit("bytes")
	if _, err := pb.FromReader(resp.Body, &buf); err != nil {
		return err
	}
	pb.Done()

	b := buf.Bytes()
	var files []archive.File
	if ok, _ := archive.IsArchive(b); ok {
		files, err = archive.Unpack(&buf)
		if err != nil {
			return err
		}
	} else {
		files = append(files, archive.File{Name: filepath.Base(self), Body: b})
	}

	if err := os.Rename(self, self+"~"); err != nil {
		return err
	}
	path := filepath.Dir(self)

	for _, i := range s.Options.RemoveBeforeUpdate {
		s.Printf("Removing %s", i)
		if err := os.RemoveAll(filepath.Join(path, i)); err != nil {
			s.Print(err)
		}
	}

Loop:
	for _, file := range files {
		for _, pattern := range s.Options.ExcludeFiles {
			matched, err := filepath.Match(pattern, file.Name)
			if err != nil {
				return err
			}
			if matched {
				continue Loop
			}
		}

		target := filepath.Join(path, file.Name)
		if file.IsDir {
			dir, err := os.Stat(target)
			if err != nil {
				if os.IsNotExist(err) {
					s.Printf("Creating directory %s", target)
					if err := os.MkdirAll(target, 0755); err != nil {
						return err
					}
				} else {
					return err
				}
			} else if !dir.IsDir() {
				return fmt.Errorf("cannot create directory %q: file exists", target)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			s.Printf("Updating file %s", target)
			if err := os.WriteFile(target, file.Body, 0644); err != nil {
				return err
			}
		}
	}

	if err := os.Chmod(self, 0755); err != nil {
		return err
	}

	if err := s.Restart(); err != nil {
		return err
	}

	if _, err := os.Stat(self); err == nil {
		return os.Remove(self + "~")
	}

	return nil
}

// Test tests the service.
func (s *Service) Test() (err error) {
	if s.Logger == nil {
		s.Logger = log.Default()
	}
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

// Command runs service command.
func (s *Service) Command(cmd string) (bool, error) {
	var err error
	switch strings.ToLower(cmd) {
	case "run":
		err = s.Run()
	case "kill":
		err = s.Kill()
	case "test":
		err = s.Test()
	case "install":
		err = s.Install()
	case "uninstall", "remove":
		err = s.Uninstall()
	case "start":
		err = s.Start()
	case "stop":
		err = s.Stop()
	case "restart":
		err = s.Restart()
	case "status":
		err = s.Status()
	case "update":
		err = s.Update()
	default:
		return false, nil
	}
	return true, err
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
