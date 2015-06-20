# dirwatcher
Show changes in directory

## Usage
```go
watcher := dirwatcher.Init(dirwatcher.Options{Showstat: 10, Logfile: "./log"})
watcher.AddDir(".")
watcher.Run()
```

```go
dirname := "."
watcher := dirwatcher.Init(dirwatcher.Options{Showstat: 10})
watcher.AddDir(dirname)
watcher.AddTrigger(testTrigger)
watcher.AddTrigger(testTrigger2)
watcher.Run()
```
