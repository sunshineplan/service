package service

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)

Loop:
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for _, pattern := range s.Options.ExcludeFiles {
			matched, err := filepath.Match(pattern, header.Name)
			if err != nil {
				return err
			}
			if matched {
				continue Loop
			}
		}

		target := filepath.Join(path, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
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
		case tar.TypeReg:
			log.Printf("Updating file %s", target)
			f, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()

		default:
			log.Printf(
				"ExtractTarGz: uknown type: %v in %s",
				header.Typeflag,
				header.Name)
		}
	}

	return s.Restart()
}
