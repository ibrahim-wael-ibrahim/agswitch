BINARY ?= agswitch
BINDIR ?= $(HOME)/.local/bin
.PHONY: fmt vet test build install clean
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
vet:
	go vet ./...
test:
	go test ./...
build:
	go build -trimpath -o $(BINARY) .
install: build
	install -d -m 0755 $(BINDIR)
	install -m 0755 $(BINARY) $(BINDIR)/$(BINARY)
clean:
	rm -f $(BINARY)
