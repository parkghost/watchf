Watchf(v0.1.3)
-------

*Watchf is a tool to execute commands when file changes*

Installation
-------
1. [install Go into your environment](http://golang.org/doc/install) 
2. install watchf

```
go get github.com/parkghost/watchf
go build github.com/parkghost/watchf
sudo mv watchf /usr/bin/watchf
```

Usage
-------

```
Usage:
  watchf options 'pattern'
Options:
  -c=[]: Add arbitrary command(repeatable)
  -s=false: To stop the watchf Daemon(windows is not support)
  -t=100ms: The time sensitive for avoid execute command frequently(time unit: ns/us/ms/s/m/h)
  -v=false: show version
Variables:
  $f: The filename of changed file
Example 1:
  watchf -c 'go vet' -c 'go test' -c 'go install' '*.go'
Example 2(Daemon):
  watchf -c 'chmod 644 $f' '*.exe' &
  watchf -s
```

Limitations
-------
1. watching changes in subdirectory
2. execute command with pipline 

Author
-------

**Brandon Chen**

+ http://brandonc.me
+ http://github.com/parkghost

License
---------------------

This project is licensed under the MIT license