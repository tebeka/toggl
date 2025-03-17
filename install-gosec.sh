#!/bin/bash

set -euo pipefail

version=$(curl -sL https://api.github.com/repos/securego/gosec/releases/latest | jq -r .name)
if [ -z "$version" ]; then
	echo "error: can't get version" 1>&2
	exit 1
fi

# v2.8.1 -> 2.8.1
version=${version#v}
echo "installing gosec $version"

os=$(go env GOOS)
arch=$(go env GOARCH)

url="https://github.com/securego/gosec/releases/download/v${version}/gosec_${version}_${os}_${arch}.tar.gz"
curl -sL "$url" | tar -C "$(go env GOPATH)/bin" -xz gosec
