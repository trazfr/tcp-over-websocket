build:
	go build -o tcptunnel *.go
build-win:
	GOOS=windows GOARCH=amd64 go build -o tcptunnel.exe *.go
build-mac:
	GOOS=darwin GOARCH=amd64 go build -o tcptunnel-amd-darwin *.go
.PHONY: build, build-win