Watchf(v0.1.8)
-------

*Watchf is a tool to watching directory for changes and execute command*

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
  watchf options
Options:
  -V=false: show debugging message
  -c=[]: Add arbitrary command (repeatable)
  -e=".*": File name matches regular expression pattern (perl-style)
  -i=0: The interval limit the frequency of the execution of commands, if equal to 0, there is no limit (time unit: ns/us/ms/s/m/h)
  -s=false: To stop the watchf Daemon (windows is not support)
  -v=false: show version
Variables:
  $f: The filename of changed file
  $t: The event type of file changes (event type: CREATE/MODIFY/DELETE)
  
Example 1:
  watchf -c 'go vet' -c 'go test' -c 'go install' -e '\.go$'
Example 2(Daemon):
  watchf -c 'process.sh $f $t' -e '\.txt$' &
  watchf -s
```

Limitations
-------
1. execute command after file closed

Author
-------

**Brandon Chen**

+ http://brandonc.me
+ http://github.com/parkghost

License
---------------------

This project is licensed under the MIT license