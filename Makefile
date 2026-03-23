.PHONY: build clean test

build:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/darwin-arm64/statusline ./src
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/darwin-amd64/statusline ./src
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/linux-arm64/statusline ./src
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/linux-amd64/statusline ./src

test:
	go test ./src/ -v

clean:
	rm -rf bin/darwin-arm64 bin/darwin-amd64 bin/linux-arm64 bin/linux-amd64
