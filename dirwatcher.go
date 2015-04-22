package dirwatcher
import
(
	"fmt"
	"os"
	"io/ioutil"
	"sync"
	"time"
	"log"
)


//TODO: Append http for view
type 
(
	taskfunc func (string, *DirWatcher)
)

/* Struct for define, when trigger happens. */
type Event struct {
	Changing bool
	Append bool
	Remove bool
}

type EventData struct {
	 trigger taskfunc
	 event Event
}

type DirWatcher struct{
	//All directories for watching
	dirs [] string
	//All files durning watching
	changes map[string]time.Time
	//All directories dyrning watching
	dirchanges map[string]time.Time
	exceptions map[string]bool
	triggers []EventData
	mutex sync.Mutex
	isstarted []bool
	//If Notify in Options is true
	notshowinfo bool
	tick time.Duration
	//Statistics
	stat Stat
	runstat bool
	loopstarted bool
	file *os.File
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
}


//Statistics
type Stat struct {
	total_append uint
	total_changed uint
}

func Init(opt ...Options)*DirWatcher {
	dirwatch := new(DirWatcher)
	dirwatch.dirs = []string{}
	dirwatch.exceptions = make(map[string]bool)
	dirwatch.changes = make(map[string]time.Time)
	dirwatch.dirchanges = make(map[string]time.Time)
	//dirwatch.triggers = []taskfunc{}
	dirwatch.triggers = []EventData {}
	dirwatch.isstarted = [] bool {}
	if len(opt) > 0 && opt[0].Notshowinfo == true {
		dirwatch.notshowinfo = true
	}

	/*
	Show statistics(Stat) after n seconds

	Statistics will be in this format (by default):
	2015-03-22 21:44:17.405990879 +0500 YEKT
	Total append:  1
	Total changed:  1
	*/
	if len(opt) > 0 && opt[0].Showstat > 0{
		dirwatch.runstat = true
		dirwatch.tick = (8*time.Second)
	}

	if len(opt) > 0 && opt[0].Showinitappend == true {
		fmt.Println(opt)
	}

	if len(opt) > 0 && opt[0].Logfile != ""{
		Logfile := opt[0].Logfile
		f, err := os.Create(Logfile)
		if err == nil {
			dirwatch.file = f
			dirwatch.exceptions[Logfile] = true
		}
	}
	return dirwatch
}


/* 
	Append new directory for watching
*/
func (d*DirWatcher) AddDir(path string){
	for _, name := range d.dirs{
		if(path == name) {
			return 
		}
	}
	d.dirs = append(d.dirs, path)
	d.isstarted = append(d.isstarted, false)
	//d.isstarted[0] = false
}


/*
	Append new traigger. (For example, do it something, when has changed specific file)

*/

func (d*DirWatcher) AddTrigger(somefunc taskfunc,  event ... Event){
	for _, funcaddr := range d.triggers {
		if(&somefunc == &funcaddr.trigger){
			return 
		}
	}
	d.triggers = append(d.triggers, EventData {somefunc, event[0]})
}

/*
	Start working of dirwatcher
*/
func (d*DirWatcher) Run(){
	if d.runstat == true {
		go d.tickEvery()
	}

	if(len(d.dirs) == 0) {
		panic("Not found directory for watching")
	}
	fmt.Println("Start dirwatcher")
	d.loopstarted = true
	for {
		for i ,dir:= range d.dirs {
			d.getAllFromDir(dir, i)
		}
		if(!d.loopstarted) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

//Stop the main loop
func (d*DirWatcher) Stop (){
	d.loopstarted = false
}

func CreateDir (path string){
	os.Mkdir(path, 0777)
}

func (d*DirWatcher) getAllFromDir(path string, i int){
	files, err := ioutil.ReadDir(path)
	if err != nil{
		log.Fatal("Cant find target dir")
	}
    for _, f := range files {
    		fullpath := path + "/" + f.Name()
            switch{
            case f.IsDir():
            	d.dirchanges[fullpath] = f.ModTime()
            default:
            	name := f.Name()
            	info := f.ModTime()
            	item, ok := d.changes[fullpath]
            	//Fullpath for this file
            	if !ok{
            		if d.isstarted[i]{
            			d.mutex.Lock()
            			d.stat.total_append += 1
            			d.mutex.Unlock()
            			d.checkTriggers(name, Event {Append: true})
            			d.showInfo("This file was append: "+ fullpath)
            		}
            		d.changes[fullpath] = info
            	} else{
            		_, errcon := d.exceptions[fullpath]
            		if item != info && !errcon{
            			d.showInfo("This file is changed: "+ fullpath)
            			d.changes[fullpath] = info
            			d.checkTriggers(name, Event {Changing: true})
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

/*
	We can only manage "system" messages(append, remove...). It not contain triggers
*/
func (d*DirWatcher) showInfo(msg string){
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

func (d*DirWatcher) checkTriggers(path string, typedata Event){
	for _, value := range d.triggers{
		d.mutex.Lock()
		if value.event.Changing == true && typedata.Changing == true{
			go value.trigger(path, d)
		}
		if value.event.Remove == true && typedata.Remove == true {
			go value.trigger(path, d)
		}
		d.mutex.Unlock()
	}
}

func (d*DirWatcher) tickEvery(){
	for i := range time.Tick(d.tick){
		go func(){
			d.mutex.Lock()
			fmt.Println(i)
			fmt.Println("Total append: ", d.stat.total_append)
			fmt.Println("Total changed: ", d.stat.total_changed)
			d.mutex.Unlock()
		}()
	}
}
