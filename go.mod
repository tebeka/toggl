module github.com/tebeka/toggl

go 1.24

tool (
	github.com/caarlos0/svu
	github.com/securego/gosec/v2/cmd/gosec
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
)

require github.com/lithammer/fuzzysearch v1.1.8

require golang.org/x/text v0.9.0 // indirect
