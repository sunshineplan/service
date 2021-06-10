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

var defaultName = "Service"

// Service represents a windows service.
type Service struct {
	Name    string
	Desc    string
	Exec    func()
	Options Options
}

// Options is Service options
type Options struct {
	Dependencies []string
	Arguments    []string
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

	resp, err := http.Get(s.Options.UpdateURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	total, err := strconv.Atoi(resp.Header.Get("content-length"))
	if err != nil {
		return err
	}

	var b bytes.Buffer
	pb := progressbar.New(total).SetUnit("bytes")
	if _, err := pb.FromReader(resp.Body, &b); err != nil {
		return err
	}
	<-pb.Done

	files, err := archive.Unpack(&b)
	if err != nil {
		return err
	}

	self, err := os.Executable()
	if err != nil {
		return err
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
			f, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := f.Write(file.Body); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
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
