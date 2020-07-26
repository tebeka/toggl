# toggle - Simple toggl command line

This is a simple command line to start/stop timers on
[toggl](https://toggl.com/). I scratched an itch, hope you'll find it useful.

## Usage

    $ toggl -h
    usage: toggl start <project>|stop|status|projects|report <since>
	    <project> - project name
	    <since> - YYYY-MM-DD (default to start of today)
      -version
	    show version and exit


You'll need a `~/.togglrc` with your API key and workspace id. See an example
[here](togglrc-example).

## Installing

If you have the Go SDK then

    go get github.com/tebeka/toggl

Or you can download a binary from the [release
section](https://github.com/tebeka/toggl/releases).

## Licence
[BSD](LICENCE)

## Contact, Bugs ...

https://github.com/tebeka/toggl
