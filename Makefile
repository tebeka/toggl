version := $(shell egrep -o '[0-9]+\.[0-9]+\.[0-9]+' toggl.go)
os=$(shell go env GOOS)
arch=$(shell go env GOARCH)

build:
	@echo Building version $(version)
	GOOS=darwin go build
	bzip2 -c toggl > toggl-$(version)-darwin-$(arch).bz2
	go build
	bzip2 -c toggl > toggl-$(version)-$(os)-$(arch).bz2
