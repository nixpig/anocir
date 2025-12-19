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
build: build-oci build-shim

.PHONY: build-oci
build-oci:
	CGO_ENABLED=1 go build -o tmp/bin/anocir cmd/anocir/main.go

.PHONY: build-shim
build-shim:
	CGO_ENABLED=1 go build -o tmp/bin/containerd-shim-anocir-v0 cmd/containerd-shim-anocir-v0/main.go

.PHONY: test
test: 
	go test -v -race ./...

.PHONY: coverage
coverage:
	go test -v -race -buildvcs -covermode atomic -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: coveralls
coveralls:
	go run github.com/mattn/goveralls@latest -coverprofile=coverage.out -service=github -repotoken=$(COVERALLS_TOKEN)

.PHONY: clean
clean:
	rm -rf tmp
	go clean

.PHONY: install
install:
	go mod download

.PHONY: vhs
vhs:
	vhs demo.tape
