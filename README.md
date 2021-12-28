# toggle - Simple toggl command line

[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/tebeka/toggle) [![Test](https://github.com/tebeka/toggl/workflows/Test/badge.svg)](https://github.com/tebeka/toggl/actions) [![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

This is a simple command line to start/stop timers on
[toggl](https://toggl.com/). I scratched an itch, hope you'll find it useful.

## Usage

    $ toggl -h
    usage: toggl start <project>|stop|status|projects|report <since>
	    <project> - project name
	    <since> - YYYY-MM-DD (default to start of today)
      -version
	    show version and exit


You'll need a `~/.togglrc` with your API key and workspace id. See an example [here](togglrc-example) (`timeout` is optional).

## Installing

If you have the Go SDK then

    go get github.com/tebeka/toggl

Or you can download a binary from the [release section](https://github.com/tebeka/toggl/releases).

## Licence
[BSD](LICENCE)

## Contact, Bugs ...

https://github.com/tebeka/toggl
