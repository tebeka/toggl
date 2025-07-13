default:
	$(error please pick a target)

test: lint
	go test -v ./...

lint:
	go tool staticcheck ./...
	go tool govulncheck ./...
	go tool gosec --terse --fmt golint ./...

release-patch:
	git tag $(shell go tool svu patch)
	git push --tags

release-minor:
	git tag $(shell go tool svu minor)
	git push --tags
