package Directory

import (
	"os"
	"path/filepath"
)

type Ut struct{}

func (f *Ut) GetPath() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return path, err
}

func (f *Ut) ParentPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Dir(wd)
}
func (f *Ut) CurentExe() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return path
}
func (f *Ut) ExistPath(p string) bool {
	stat, err := os.Stat(p)
	if err == nil && stat.IsDir() {
		return true
	}
	return false
}

func (f *Ut) CreatePath(p string) (bool, error) {
	err := os.Mkdir(p, 7777)
	if err == nil {
		return true, nil
	}
	return false, err
}
