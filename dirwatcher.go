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

type DirWatcher struct{
	dirs [] string
	changes map[string]time.Time
	dirchanges map[string]time.Time
	triggers []taskfunc
	mutex sync.Mutex
	isstarted bool
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
	dirwatch.triggers = []taskfunc{}
	dirwatch.isstarted = false
	if len(opt) > 0 && opt[0].Notshowinfo == true {
		dirwatch.notshowinfo = true
	}

	if len(opt) > 0 && opt[0].Showstat > 0{
		dirwatch.runstat = true
		dirwatch.tick = (8*time.Second)
	}
	return dirwatch
}


/* 
	Append new directory for watching
*/
func (d*DirWatcher) AddDir(path string){
	d.dirs = append(d.dirs, path)
}


/*
	Append new Task
*/
func (*DirWatcher) AddTask(path string){

}

/*
	Append new traigger. (For example, do it something, when has changed specific file)

*/

func (d*DirWatcher) AddTrigger(somefunc taskfunc){
	d.triggers = append(d.triggers, somefunc)
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
		for _ ,dir:= range d.dirs {
			d.getAllFromDir(dir)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func CreateDir (path string){
	os.Mkdir(path, 0777)
}

func (d*DirWatcher) getAllFromDir(path string){
	files, err := ioutil.ReadDir(path)
	if err != nil{
		log.Fatal("Cant find target dir")
	}
    for _, f := range files {
            switch{
            case f.IsDir():
            	d.dirchanges[f.Name()] = f.ModTime()
            default:
            	name := f.Name()
            	info := f.ModTime()
            	item, ok := d.changes[name]
            	if !ok{
            		if d.isstarted{
            			d.mutex.Lock()
            			d.stat.total_append += 1
            			d.mutex.Unlock()
            			d.showInfo("This file was append: "+ name)
            		}
            		d.changes[name] = info
            	} else{
            		if item != info{
            			d.showInfo("This file is changed: "+ name)
            			d.changes[name] = info
            			d.checkTriggers(name)
            			d.mutex.Lock()
            			d.stat.total_changed += 1
            			d.mutex.Unlock()
            		}
            	}

            }
    }
    d.isstarted = true
}

/*
	We can only manage "system" messages(append, remove...). It not contain triggers
*/
func (d*DirWatcher) showInfo(msg string){
	if d.notshowinfo != true {
		fmt.Println(msg)
	}
}

func (d*DirWatcher) checkTriggers(path string){
	for _, trigger := range d.triggers{
		d.mutex.Lock()
		go trigger(path)
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
