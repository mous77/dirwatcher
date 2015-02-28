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
}


func Init()*DirWatcher {
	dirwatch := new(DirWatcher)
	dirwatch.dirs = []string{}
	dirwatch.changes = make(map[string]time.Time)
	dirwatch.dirchanges = make(map[string]time.Time)
	dirwatch.triggers = []taskfunc{}
	dirwatch.isstarted = false
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
            			fmt.Println("This file was append: ", name)
            		}
            		d.changes[name] = info
            	} else{
            		if item != info{
            			fmt.Println("This file is changed: ", name)
            			d.changes[name] = info
            			d.checkTriggers(name)
            		}
            	}

            }
    }
    d.isstarted = true
}

func (d*DirWatcher) checkTriggers(path string){
	for _, trigger := range d.triggers{
		d.mutex.Lock()
		go trigger(path)
		d.mutex.Unlock()
	}
}
