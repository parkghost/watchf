Watchf
-------
[![Build Status](https://travis-ci.org/parkghost/watchf.png)](https://travis-ci.org/parkghost/watchf)

*Watchf is a tool for watching directory changes and run commands*

Screenshot
-------
![Screenshot](http://i.imgur.com/blF6Hh7.png)

Installation
-------
1. [install Go into your environment](http://golang.org/doc/install) 
2. install watchf

```
go get github.com/parkghost/watchf/...
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
  -exclude="^\.": Do not process any events whose file name matches specified regular expression pattern (perl-style)
  -f=".watchf.conf": Specifies a configuration file
  -i=100ms: The interval limit the frequency of the command executions, if equal to 0, there is no limit (time unit: ns/us/ms/s/m/h)
  -include=".*": Process any events whose file name matches file name matches specified regular expression pattern (perl-style)
  -r=false: Watch directories recursively
  -w=false: Write command-line arguments to configuration file (write and exit)

Events:
     all  Create/Write/Remove/Rename/Chmod
  create  File/directory created in watched directory
  write   File written in watched directory
  remove  File/directory deleted from watched directory
  rename  File moved out of watched directory
  chmod   File Metadata changed

Variables:
      %f  The filename of changed file
      %t  The event type of file changes

Example 1:
  watchf -e "write,remove,create" -c "go test" -c "go vet" -include ".go$"
Example 2(with custom variable):
  watchf -c "process.sh %f %t"
Example 3(with configuration file):
  watchf -e "write,remove,create" -c "go test" -c "go vet" -include ".go$" -w
  watchf
```

Binaries
-------
[Link](https://github.com/parkghost/watchf/releases)

License
---------------------

This project is licensed under the MIT license
