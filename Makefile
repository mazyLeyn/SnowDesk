LDFLAGS = -H=windowsgui -s -w

.PHONY: all test preview clean
all: dist/snowdesk-windows-amd64.exe dist/snowdesk-windows-arm64.exe

dist/snowdesk-windows-%.exe:
	GOOS=windows GOARCH=$* CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $@ ./cmd/snowdesk

test:
	go test ./snow/
	GOOS=windows go vet ./cmd/snowdesk/

preview:
	go run ./cmd/preview -out docs/preview.png -icon docs/icon.png

clean:
	rm -rf dist
