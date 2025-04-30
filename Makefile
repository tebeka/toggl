default:
	$(error please pick a target)

test: lint
	go test -v ./...

lint:
	staticcheck ./...
	govulncheck ./...
	gosec --terse --fmt golint ./...

release-patch:
	git tag $(shell svu patch)
	git push --tags

release-minor:
	git tag $(shell svu minor)
	git push --tags
