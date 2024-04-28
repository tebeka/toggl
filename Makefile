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
	curl -L https://github.com/securego/gosec/releases/download/v2.19.0/gosec_2.19.0_linux_amd64.tar.gz | \
		tar -C $(shell go env GOPATH)/bin -xz gosec
