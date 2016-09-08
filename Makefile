version := $(shell egrep -o '[0-9]+\.[0-9]+\.[0-9]+' toggl.go)
os=$(shell go env GOOS)
arch=$(shell go env GOARCH)

build:
	go build
	bzip2 -c toggl > toggl-$(version)-$(os)-$(arch).bz2

