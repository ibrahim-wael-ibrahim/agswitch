BINARY ?= agswitch
BUILDDIR ?= bin
BINDIR ?= $(HOME)/.local/bin
VERSION ?= dev
LDFLAGS := -s -w -X github.com/ibrahim-wael/agswitch/cmd.version=$(VERSION)
OUTPUT := $(BUILDDIR)/$(BINARY)

.PHONY: fmt tidy vet test race build install uninstall release clean check

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

build:
	mkdir -p $(BUILDDIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUTPUT) .

install: build
	install -d -m 0755 $(BINDIR)
	install -m 0755 $(OUTPUT) $(BINDIR)/$(BINARY)

uninstall:
	rm -f $(BINDIR)/$(BINARY)

release:
	rm -rf $(BUILDDIR)/release
	mkdir -p $(BUILDDIR)/release
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILDDIR)/release/agswitch_linux_amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILDDIR)/release/agswitch_linux_arm64 .
	cd $(BUILDDIR)/release && sha256sum agswitch_linux_* > checksums.txt

check: fmt tidy vet test build

clean:
	rm -rf $(BUILDDIR)
