package dirwatcher

import (
	"testing"
	//"sync"
	"os"
	"time"
)

const (
	dirdata = "./testdata/"
)

func createFile(path string) error {
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}
	f.Write([]byte("123"))
	return nil
}

func modifyFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0660)
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
		t.Errorf("%v", err)
	}
	defer os.Remove(path)

	watcher := Init()
	complete := false
	watcher.AddFile(path, func(item string, d *DirWatcher) {
		if path == item {
			complete = true
		}
	})
	boo := make(chan bool, 1)
	go func() {
		watcher.Run()
	}()
	go func() {
		time.Sleep(time.Second * 2)
		boo <- true
	}()

	modifyFile(path)
	tickChan := time.NewTicker(time.Millisecond * 400).C
	for {
		select {
		case <-tickChan:
			if complete {
				return
			}
		case <-boo:
			{
				t.Errorf("Fail %d", complete)
				return
			}
		}
	}
}
