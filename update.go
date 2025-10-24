package service

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

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
	selfTmp := fmt.Sprintf("%s.%s.tmp", self, time.Now().Format(time.DateOnly))

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
	pb.Wait()

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
	path := filepath.Dir(self)

	s.Printf("Stopping %s", s.Name)
	if err := s.Stop(); err != nil {
		return err
	}

	for _, i := range s.Options.RemoveBeforeUpdate {
		if file := filepath.Join(path, i); file != self {
			s.Printf("Removing %s", i)
			if err := os.RemoveAll(file); err != nil {
				s.Print(err)
			}
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
		stat, err := os.Stat(target)
		if file.IsDir {
			if err != nil {
				if os.IsNotExist(err) {
					s.Printf("Creating directory %s", target)
					if err := os.MkdirAll(target, 0755); err != nil {
						return err
					}
				} else {
					return err
				}
			} else if !stat.IsDir() {
				return fmt.Errorf("cannot create directory %q: file exists", target)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			s.Printf("Updating file %s", target)
			if target == self {
				if err := os.Rename(self, selfTmp); err != nil {
					return err
				}
			}
			var perm os.FileMode
			if stat != nil {
				perm = stat.Mode().Perm()
			} else {
				perm = 0644
			}
			if err := os.WriteFile(target, file.Body, perm); err != nil {
				return err
			}
		}
	}
	s.Printf("Starting %s", s.Name)
	if runtime.GOOS == "darwin" {
		if err := s.reload(); err != nil {
			return err
		}
	} else {
		if err := s.Start(); err != nil {
			return err
		}
	}
	if _, err := os.Stat(selfTmp); err == nil {
		return os.Remove(selfTmp)
	}
	return nil
}
