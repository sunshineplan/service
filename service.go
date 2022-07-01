package service

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sunshineplan/utils/archive"
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
  update
    	Update service files if update url is provided
`

var defaultName = "Service"

// Service represents a windows service.
type Service struct {
	Name     string
	Desc     string
	Exec     func()
	TestExec func() error
	Options  Options
}

// Options is Service options
type Options struct {
	Dependencies []string
	Arguments    []string
	Environment  map[string]string
	Others       []string
	UpdateURL    string
	ExcludeFiles []string
}

// New creates a new service name.
func New() *Service {
	return &Service{Name: defaultName}
}

// Update updates the service's installed files.
func (s *Service) Update() error {
	if s.Options.UpdateURL == "" {
		return fmt.Errorf("no update url provided")
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
					log.Printf("Creating dir %s", target)
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

			log.Printf("Updating file %s", target)
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
	if s.TestExec != nil {
		err = s.TestExec()
		if err != nil {
			log.Println("Test failed:", err)
		} else {
			log.Print("Test pass.")
		}
	} else {
		log.Print("No test provided.")
	}
	return nil
}

// Remove is an alias for Uninstall.
func (s *Service) Remove() error {
	return s.Uninstall()
}
