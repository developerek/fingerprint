#!/bin/zsh



go mod init fingerprint
go mod tidy
# Check for errors and warnings
go vet ./... || exit

# Format your code
go fmt ./... || exit

# Format the code
dirs=$(go list -f {{.Dir}} ./...)
for d in $dirs; do goimports -w $d/*.go; done
