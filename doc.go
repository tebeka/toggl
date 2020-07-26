/* toggle - Simple toggl command line

This is a simple command line to start/stop timers on https://toggl.com.
I scratched an itch, hope you'll find it useful.

Usage

    $ toggl -h
    usage: toggl start <project>|stop|status|projects|report <since>
	    <project> - project name
	    <since> - YYYY-MM-DD (default to start of today)
      -version
	    show version and exit


You'll need a `~/.togglrc` with your API key and workspace id. See
`togglrc-example` in the github repo.
*/

package main
