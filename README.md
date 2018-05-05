# toggle - Simple toggl command line

This is a simple command line to start/stop timers on
[toggl](https://toggl.com/). I scratched an itch, hope you'll find it useful.

## Usage

    toggl -h
    usage: toggl start <project>|stop|status|projects|entries|week
      -version
            show version and exit

#### Examples

    toggl start --at="5 minutes ago" --stop="in 20 minutes" MyProject
    toggl start --at="10:00" --stop="11:00" --description="Working on toggl" MyProject
    toggl status
    toggl entries
    toggl week

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
