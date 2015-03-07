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
	taskfunc func (string)
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
	dirs [] string
	changes map[string]time.Time
	dirchanges map[string]time.Time
	triggers []EventData
	mutex sync.Mutex
	isstarted []bool
	//If Notify in Options is true
	notshowinfo bool
	tick time.Duration
	//Statistics
	stat Stat
	runstat bool
}


type Options struct {
	//Show messages about append, modify etc
	Notshowinfo bool

	//Show statistics every n seconds
	Showstat uint

	//Show initial append files
	Showinitappend bool
}


//Statistics
type Stat struct {
	total_append uint
	total_changed uint
}

func Init(opt ...Options)*DirWatcher {
	dirwatch := new(DirWatcher)
	dirwatch.dirs = []string{}
	dirwatch.changes = make(map[string]time.Time)
	dirwatch.dirchanges = make(map[string]time.Time)
	//dirwatch.triggers = []taskfunc{}
	dirwatch.triggers = []EventData {}
	dirwatch.isstarted = [] bool {}
	if len(opt) > 0 && opt[0].Notshowinfo == true {
		dirwatch.notshowinfo = true
	}

	if len(opt) > 0 && opt[0].Showstat > 0{
		dirwatch.runstat = true
		dirwatch.tick = (8*time.Second)
	}

	if len(opt) > 0 && opt[0].Showinitappend == true {
		fmt.Println(opt)
	}
	return dirwatch
}


/* 
	Append new directory for watching
*/
func (d*DirWatcher) AddDir(path string){
	d.dirs = append(d.dirs, path)
	d.isstarted = append(d.isstarted, false)
	//d.isstarted[0] = false
}


/*
	Append new traigger. (For example, do it something, when has changed specific file)

*/

func (d*DirWatcher) AddTrigger(somefunc taskfunc,  event ... Event){
	d.triggers = append(d.triggers, EventData {somefunc, event[0]})
}

/*
	Start working of dirwatcher
*/
func (d*DirWatcher) Run(){
	if d.runstat == true {
		go d.tickEvery()
	}
	fmt.Println("Start dirwatcher")
	for {
		for i ,dir:= range d.dirs {
			d.getAllFromDir(dir, i)
		}
		time.Sleep(100 * time.Millisecond)
	}
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
            		if item != info{
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
}

func (d*DirWatcher) checkTriggers(path string, typedata Event){
	for _, value := range d.triggers{
		d.mutex.Lock()
		if value.event.Changing == true && typedata.Changing == true{
			go value.trigger(path)
		}
		if value.event.Remove == true && typedata.Remove == true {
			go value.trigger(path)
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
