package dirwatcher

import (
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/op/go-logging"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var log = logging.MustGetLogger("lightstore_log")

//TODO: Append http for view
type (
	taskfunc func(string, *DirWatcher)
)

//Struct for define, when trigger happens.
type Event struct {
	Changing bool
	Append   bool
	Remove   bool
}

type EventData struct {
	trigger taskfunc
	event   Event
}

type DirWatcher struct {
	//All directories for watching
	dirs []string
	//All files durning watching
	changes map[string]time.Time
	//All directories dyrning watching
	dirchanges map[string]time.Time
	exceptions map[string]bool
	triggers   []EventData
	mutex      sync.Mutex
	isstarted  []bool
	//If Notify in Options is true
	notshowinfo bool
	tick        time.Duration
	//Statistics
	stat        Stat
	runstat     bool
	loopstarted bool
	file        *os.File
	//directory for backup files
	backupdir string
	server    bool

	//This command stops main loop
	stop bool

	//All file names in directories
	allfilenames     []string
	currentfilenames []string

	recursive    bool
	ignorehidden bool
}

type Options struct {
	//Show messages about append, modify etc
	Notshowinfo bool

	//Show statistics every n seconds
	Showstat uint

	//Show initial append files
	Showinitappend bool

	//Log for output
	Logfile string

	//Directory for backup files
	Backup string

	//Start server(in this stage, run server with default options)
	Server bool

	//This option provides wathing subdirectories
	Recursive bool

	//Ignore changes on hidden files
	IgnoreHidden bool
}

//Statistics
type Stat struct {
	total_append  uint
	total_changed uint
	total_remove  uint
}

//DirWatcherRequest provides actions after POST
//action can be [add, remove]
type DirWatcherRequest struct {
	Path   string
	Action string
}

//Init provides basic initialization
func Init(opt ...Options) *DirWatcher {
	dirwatch := new(DirWatcher)
	dirwatch.dirs = []string{}
	dirwatch.exceptions = make(map[string]bool)
	dirwatch.changes = make(map[string]time.Time)
	dirwatch.dirchanges = make(map[string]time.Time)
	//dirwatch.triggers = []taskfunc{}
	dirwatch.triggers = []EventData{}
	dirwatch.isstarted = []bool{}
	dirwatch.stop = false

	if len(opt) == 0 {
		return dirwatch
	}
	if opt[0].Notshowinfo == true {
		dirwatch.notshowinfo = true
	}

	/*
		Show statistics(Stat) after n seconds

		Statistics will be in this format (by default):
		2015-03-22 21:44:17.405990879 +0500 YEKT
		Total append:  1
		Total changed:  1
	*/
	if opt[0].Showstat > 0 {
		dirwatch.runstat = true
		dirwatch.tick = (8 * time.Second)
	}

	if opt[0].Showinitappend == true {
		fmt.Println(opt)
	}

	if opt[0].Logfile != "" {
		Logfile := opt[0].Logfile
		f, err := os.Create(Logfile)
		if err == nil {
			dirwatch.file = f
			dirwatch.exceptions[Logfile] = true
		}
	}

	if opt[0].Backup != "" {
		dirwatch.backupdir = opt[0].Backup
		_, err := os.Stat(dirwatch.backupdir)
		if err != nil {
			os.Mkdir(dirwatch.backupdir, 0777)
		}
	}

	if opt[0].Server == true {
		dirwatch.server = true
	}

	if opt[0].Recursive == true {
		dirwatch.recursive = true
	}

	if opt[0].IgnoreHidden == true {
		dirwatch.ignorehidden = true
	}
	return dirwatch
}

//Copy files to backup dir before starting of watching
func (d *DirWatcher) copyToBackup() {
	fmt.Println("Copt files to backup dir(**Test version**)")
	for _, dirname := range d.dirs {
		files, err := ioutil.ReadDir(dirname)
		if err != nil {
			log.Fatal("Cant find target dir")
		}

		for _, pathvalue := range files {
			fullpath := dirname + "/" + pathvalue.Name()
			if checker, _ := os.Stat(fullpath); checker.IsDir() {
				continue
			}
			filedata, _ := os.Open(fullpath)
			backupfile, err := os.Create(d.backupdir + "/" + path.Base(pathvalue.Name()))
			if err != nil {
				panic(err)
			}
			_, errcopy := io.Copy(backupfile, filedata)
			if errcopy != nil {
				panic(err)
			}
		}
	}
}

//Append new directory for watching
func (d *DirWatcher) AddDir(path string) {
	for _, name := range d.dirs {
		if path == name {
			return
		}
	}
	d.dirs = append(d.dirs, path)
	d.isstarted = append(d.isstarted, false)
	//d.isstarted[0] = false
}

//removeDir, works only with POST request
func (d *DirWatcher) removeDir(path string) {
	for i, name := range d.dirs {
		if path == name {
			d.dirs = append(d.dirs[:i], d.dirs[i+1:]...)
			return
		}
	}
}

//AddTrigger provides append new traigger.
//(For example, do it something, when has changed specific file)
func (d *DirWatcher) AddTrigger(somefunc taskfunc, event Event) {
	for _, funcaddr := range d.triggers {
		if &somefunc == &funcaddr.trigger {
			return
		}
	}
	d.triggers = append(d.triggers, EventData{somefunc, event})
}

//showDirs shows
func (d *DirWatcher) showDirs() {
	fmt.Println("Watching directories:")
	for _, dir := range d.dirs {
		fmt.Println(dir)
	}
}

//Run is start working of dirwatcher
func (d *DirWatcher) Run() {
	if d.runstat == true {
		go d.tickEvery()
	}

	if len(d.dirs) == 0 {
		panic("Not found directory for watching")
	}

	if d.backupdir != "" {
		d.copyToBackup()
	}

	d.showDirs()
	fmt.Println("Start dirwatcher")
	d.loopstarted = true
	if d.server {
		go d.runServer()
	}

	for {
		if d.stop {
			break
		}

		for i, dir := range d.dirs {
			d.getAllFromDir(dir, i)
			if len(d.allfilenames) == 0 {
				d.allfilenames = d.currentfilenames
			} else if len(d.allfilenames) > len(d.currentfilenames) {
				diff := difference(d.allfilenames, d.currentfilenames)
				fmt.Println("Removed files: ")
				for _, item := range diff {
					d.stat.total_remove += 1
					fmt.Println(item)
				}
				d.allfilenames = d.currentfilenames
			}
			d.currentfilenames = []string{}
		}
		if !d.loopstarted {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

//This function returns file names contains in allfilenames,
//but not contains in currentfilenames
func difference(allfilenames, currentfilenames []string) []string {
	diffitems := []string{}
	for _, afnitem := range allfilenames {
		found := false
		for _, cfnitem := range currentfilenames {
			if afnitem == cfnitem {
				found = true
				break
			}
		}
		if !found {
			diffitems = append(diffitems, afnitem)
		}
	}

	return diffitems
}

//RunServer starts rest server
func (d *DirWatcher) runServer() {
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	router, err := rest.MakeRouter(
		&rest.Route{"POST", "/dirwatcher", func(w rest.ResponseWriter, r *rest.Request) {
			dwreq := DirWatcherRequest{}
			report := ""
			err := r.DecodeJsonPayload(&dwreq)
			if err != nil {
				rest.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if dwreq.Path != "" && dwreq.Action == "add" {
				d.AddDir(dwreq.Path)
				report += fmt.Sprintf("Add new dir %s \n", dwreq.Path)
			}

			if dwreq.Path != "" && dwreq.Action == "remove" {
				d.removeDir(dwreq.Path)
				report += fmt.Sprintf("Remove dir %s \n", dwreq.Path)
			}

			if dwreq.Path != "" && dwreq.Action == "info" {
				stat := d.stat
				report += fmt.Sprintf("Total append %d Total Changed %d Total removed %d ",
					stat.total_append, stat.total_changed, stat.total_remove)
			}

			if dwreq.Action == "stop" {
				d.stop = true
			}
			w.WriteJson(report)
		}},
	)

	if err != nil {
		log.Fatal(err)
	}
	api.SetApp(router)
	addr := "localhost"
	port := 8080
	http.ListenAndServe(fmt.Sprintf("%s:%d", addr, port), api.MakeHandler())
}

//Stop the main loop
func (d *DirWatcher) Stop() {
	d.loopstarted = false
}

func CreateDir(path string) {
	os.Mkdir(path, 0777)
}

//getAllFromDir provides reading directory
func (d *DirWatcher) getAllFromDir(path string, i int) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal("Cant find target dir")
	}
	for _, f := range files {
		fullpath := path + "/" + f.Name()
		switch {
		case f.IsDir():
			d.dirchanges[fullpath] = f.ModTime()
		default:
			name := f.Name()
			if d.ignorehidden && strings.HasPrefix(name, ".") {
				continue
			}
			d.currentfilenames = append(d.currentfilenames, name)
			info := f.ModTime()
			item, ok := d.changes[fullpath]
			//Fullpath for this file
			if !ok {
				if d.isstarted[i] {
					d.mutex.Lock()
					d.stat.total_append += 1
					d.mutex.Unlock()
					d.checkTriggers(name, Event{Append: true})
					d.showInfo("This file was append: " + fullpath)
					d.allfilenames = append(d.allfilenames, name)
				}
				d.changes[fullpath] = info
			} else {
				_, errcon := d.exceptions[fullpath]
				if item != info && !errcon {
					d.showInfo("This file is changed: " + fullpath)
					d.changes[fullpath] = info
					d.checkTriggers(name, Event{Changing: true})
					//d.checkTriggers(fullpath)
					d.mutex.Lock()
					d.stat.total_changed += 1
					d.mutex.Unlock()
				}
			}

		}
	}
	d.isstarted[i] = true //; Note: Works only for one dir
}

//TODO. Make with recursive scanning
func (d *DirWatcher) watchSubDirs(path string) {
	allfiles := []string{}
	filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		allfiles = append(allfiles, f.Name())
		return nil
	})
}

//We can only manage "system" messages(append, remove...). It not contains triggers
func (d *DirWatcher) showInfo(msg string) {
	if d.notshowinfo != true {
		fmt.Println(msg)
	}

	if d.file != nil {
		d.mutex.Lock()
		d.file.WriteString(msg)
		d.file.WriteString("\n")
		d.mutex.Unlock()
	}
}

//checkTriggers provides checking and execution triggers
func (d *DirWatcher) checkTriggers(path string, typedata Event) {
	for _, value := range d.triggers {
		d.mutex.Lock()
		if value.event.Changing == true && typedata.Changing == true {
			go value.trigger(path, d)
		}
		if value.event.Remove == true && typedata.Remove == true {
			go value.trigger(path, d)
		}
		d.mutex.Unlock()
	}
}

//tickEvery provides show statistics every n seconds
func (d *DirWatcher) tickEvery() {
	for i := range time.Tick(d.tick) {
		go func() {
			d.mutex.Lock()
			fmt.Println(i)
			fmt.Println(fmt.Sprintf("Total append: %d", d.stat.total_append))
			fmt.Println(fmt.Sprintf("Total changed: %d", d.stat.total_changed))
			fmt.Println(fmt.Sprintf("Total removed: %d", d.stat.total_remove))
			d.mutex.Unlock()
		}()
	}
}
