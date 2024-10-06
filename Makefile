.PHONY: tidy
tidy: 
	go fmt ./...
	go mod tidy -v

.PHONY: audit
audit:
	go mod verify
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: build
build:
	go build -o tmp/bin/brownie main.go

.PHONY: test
test: 
	go run gotest.tools/gotestsum@latest ./...

.PHONY: coverage
coverage:
	go test -v -race -buildvcs -covermode atomic -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: coveralls
coveralls:
	go run github.com/mattn/goveralls@latest -coverprofile=coverage.out -service=github

.PHONY: run
run:
	go run ./...

.PHONY: watch
watch:
		go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" \
		--build.bin "tmp/bin/brownie" \
		--build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go" \
		--misc.clean_on_exit "true"

.PHONY: clean
clean:
	rm -rf tmp
	go clean

.PHONY: install
install:
	go mod download
