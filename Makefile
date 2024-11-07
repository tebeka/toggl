default:
	$(error please pick a target)

test: lint
	go test -v ./...

lint:
	staticcheck ./...
	govulncheck ./...
	gosec ./...

gosec_url=https://github.com/securego/gosec/releases/download/v2.21.4/gosec_2.21.4_linux_amd64.tar.gz

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	curl -L $(gosec_url) | tar -C $(shell go env GOPATH)/bin -xz gosec
