package service

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sunshineplan/utils/archive"
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
	Arguments    string
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
		return fmt.Errorf("No update url provided")
	}

	path, err := os.Executable()
	if err != nil {
		return err
	}

	resp, err := http.Get(s.Options.UpdateURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	files, err := archive.Unpack(resp.Body)
	if err != nil {
		return err
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
					log.Printf("Creating dir %s", target)
					if err := os.MkdirAll(target, 0755); err != nil {
						return err
					}
				} else {
					return err
				}
			} else if !dir.IsDir() {
				return fmt.Errorf("Cannot create directory %q: File exists", target)
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

	return s.Restart()
}
