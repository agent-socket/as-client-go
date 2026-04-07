default: build

build:
    go build ./...

test:
    go test ./...

vet:
    go vet ./...

clean:
    go clean ./...

tidy:
    go mod tidy
