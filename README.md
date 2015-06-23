# dirwatcher
Show changes in directory

Work in progress

Supported events: add, change, remove

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

Starts watching with server
```go
dirname := "."
watcher := dirwatcher.Init(dirwatcher.Options{Server: true})
watcher.AddDir(dirname)
watcher.Run()
```
After that, you can manage dirwatcher with post requests.

In this example, we add directory ".." for watching.
```
curl -i -H 'Content-Type: application/json' -d '{"Path":"..", "Action": "add"}' http://127.0.0.1:8080/dirwatcher
```

In this example, remove directory ".." from watching
```
curl -i -H 'Content-Type: application/json' -d '{"Path":"..", "Action": "remove"}' http://127.0.0.1:8080/dirwatcher
