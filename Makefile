BINARY ?= agswitch
BUILDDIR ?= bin
BINDIR ?= $(HOME)/.local/bin
OUTPUT := $(BUILDDIR)/$(BINARY)

.PHONY: fmt tidy vet test build install clean check

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./...

build:
	mkdir -p $(BUILDDIR)
	go build -trimpath -o $(OUTPUT) .

install: build
	install -d -m 0755 $(BINDIR)
	install -m 0755 $(OUTPUT) $(BINDIR)/$(BINARY)

check: fmt tidy vet test build

clean:
	rm -rf $(BUILDDIR)
