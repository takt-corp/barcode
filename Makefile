# process run during github actions
ci: install
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

# install dependencies
install:
	go get .
	go mod tidy

# upgrade dependencies
upgrade:
	go mod tidy
	go get -u ./...
	go mod tidy

# run lint using golangci-linters
lint:
	golangci-lint run

# run tests
test:
	go test -cover -coverprofile=coverage.out ./...
