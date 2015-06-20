# dirwatcher
Show changes in directory

## Usage
```go
watcher := dirwatcher.Init(dirwatcher.Options{Showstat: 10, Logfile: "./log"})
watcher.AddDir(".")
watcher.Run()
```:
