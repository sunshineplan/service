package service

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/sunshineplan/utils/archive"
	"github.com/sunshineplan/utils/progressbar"
)

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
	if runtime.GOOS == "darwin" {
		run("codesign", "--sign", "-", "--force", "--preserve-metadata=entitlements,requirements,flags,runtime", self)
	}

	if err := s.Restart(); err != nil {
		return err
	}

	if _, err := os.Stat(self); err == nil {
		return os.Remove(self + "~")
	}

	return nil
}
