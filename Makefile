default:
	$(error please pick a target)

test: lint
	go test -v ./...

lint:
	staticcheck ./...
	govulncheck ./...
	gosec ./...

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	./install-gosec.sh
	go install github.com/caarlos0/svu@latest

release-patch:
	git tag $(shell svu patch)
	git push --tags

release-minor:
	git tag $(shell svu minor)
	git push --tags
