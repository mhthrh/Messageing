package File

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type fileS struct {
	path     string
	fileName string
	mu       sync.Mutex
}

func New(p, f string) *fileS {
	return &fileS{fileName: f, path: p}
}

func (f *fileS) Read() (string, error) {
	//isdelve.Enabled
	//fmt.Println(f.path)
	//f.path = ""
	fmt.Println(filepath.Join(f.path, f.fileName))
	byt, err := os.ReadFile(filepath.Join(f.path, f.fileName))
	return string(byt), err
}

// Write string to fileS
func (f *fileS) Write(s string) error {
	f.mu.Lock()

	f1, err := os.Create(filepath.Join(f.path, f.fileName))
	if err != nil {
		return err
	}
	defer func() {
		f1.Close()
		f.mu.Unlock()
	}()
	_, err = f1.WriteString(s)
	//f.mu.Unlock()
	if err != nil {
		return err
	}
	f1.Sync()
	//f1.Close()
	return nil
}
