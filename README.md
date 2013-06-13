Watchf(v0.3.2)
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
  watchf [options]
Options:
  -V=false: Show debugging messages
  -c=[]: Add arbitrary command (repeatable)
  -e=[all]: Listen for specific event(s) (comma separated list)
  -f=".watchf.conf": Specifies a configuration file
  -i=0: The interval limit the frequency of the command executions, if equal to 0, there is no limit (time unit: ns/us/ms/s/m/h)
  -p=".*": File name matches regular expression pattern (perl-style)
  -r=false: Watch directories recursively
  -s=false: Stop the watchf Daemon (windows is not support)
  -v=false: Show version and exit
  -w=false: Write command-line arguments to configuration file (write and exit)
Events:
  all     Create/Delete/Modify/Rename
  create  File/directory created in watched directory
  delete  File/directory deleted from watched directory
  modify  File was modified or Metadata changed
  rename  File moved out of watched directory
Variables:
  %f: The filename of changed file
  %t: The event type of file changes
Example 1:
  watchf -e "modify,delete" -c "go vet" -c "go test" -c "go install" -p "\.go$"
Example 2(with custom variable):
  watchf -c "process.sh %f %t"
Example 3(with daemon):
  watchf -r -c "rsync -aq $SRC $DST" &
  watchf -s
Example 4(with configuration file):
  watchf -e "modify,delete" -c "go vet" -c "go test" -c "go install" -p "\.go$" -w
  watchf
```

Pre-built Binaries
-------
[http://bit.ly/19sJFdj](http://bit.ly/19sJFdj)


Author
-------

**Brandon Chen**

+ http://brandonc.me
+ http://github.com/parkghost

License
---------------------

This project is licensed under the MIT license