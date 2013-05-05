Watchf(v0.1.7)
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
  watchf options ['pattern']
Options:
  -V=false: show debugging message
  -c=[]: Add arbitrary command (repeatable)
  -s=false: To stop the watchf Daemon (windows is not support)
  -t=500ms: The time sensitive for avoid execute command frequently (time unit: ns/us/ms/s/m/h)
  -v=false: show version
Patterns:
  '*'         matches any sequence of non-Separator characters e.g. '*.txt'
  '?'         matches any single non-Separator character       e.g. 'ab?.txt'
  '[' [ '^' ] { character-range } ']'                          e.g. 'ab[b-d].txt'
              character class (must be non-empty)
   c          matches character c (c != '*', '?', '\\', '[')   e.g. 'abc.txt'
Variables:
  $f: The filename of changed file
  $t: The event type of file changes (event type: CREATE/MODIFY/DELETE/RENAME)
  
Example 1:
  watchf -c 'go vet' -c 'go test' -c 'go install' '*.go'
Example 2(Daemon):
  watchf -c 'process.sh $f $t' '*.txt' &
  watchf -s
```

Limitations
-------
1. watching changes in subdirectory
2. execute command after file closed

Author
-------

**Brandon Chen**

+ http://brandonc.me
+ http://github.com/parkghost

License
---------------------

This project is licensed under the MIT license