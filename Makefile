.PHONY: test clean build install

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf dist/
	rm -f kubectl-sniff
	rm -f kubectl-sniff-*
	rm -f ksniff.zip

# Local development build (builds for current platform only)
build:
	go build -o kubectl-sniff cmd/kubectl-sniff.go

# Install locally (for development/testing)
install: build
	mkdir -p /usr/local/bin
	cp kubectl-sniff /usr/local/bin/kubectl-sniff
