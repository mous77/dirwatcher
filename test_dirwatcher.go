package dirwatcher

import (
  "testing"
  //"sync"
  "os"
  "time"
)
const (
	dirdata = "../testdata/"
)


func createFile(path string) error {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return err
	}
	f.Write([]byte("123"))
	return nil
}

func modifyFile(path string) error {
	f, err := os.OpenFile("foo.txt", os.O_RDWR|os.O_APPEND, 0660)
	defer f.Close()
	if err != nil {
		return err
	}
	f.Write([]byte("123"))
	return nil
}


func TestWatchingFiles(t *testing.T) {
	path := dirdata + "watch1"
	err := createFile(path)
	if err != nil {
		t.Error("%v", err)
	}
	defer os.Remove(path)


	watcher := Init()
	trigged := false
	watcher.AddFile(path, func(item string, d *DirWatcher){
		trigged = true
	})
	boo := make(chan bool, 1)
	go func() {
		watcher.Run()
	}()
	go func() {
		time.Sleep(time.Second)
        boo <- true
	}()
}