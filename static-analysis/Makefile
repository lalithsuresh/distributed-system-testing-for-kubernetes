all: clean build run

clean:
	@go clean

build:
	@go build main.go

run:
	@go run main.go

test:
	@go test -v kubetorch/ssapasses/collector

format:
	@gofmt -s -w -l .
