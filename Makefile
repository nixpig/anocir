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
build: build-oci build-cri

.PHONY: build-oci
build-oci:
	CGO_ENABLED=1 go build -o tmp/bin/anocir cmd/anocir/main.go

.PHONY: build-cri
build-cri:
	CGO_ENABLED=1 go build -o tmp/bin/anocird cmd/anocird/main.go

.PHONY: test
test: 
	go test -v -race ./...

.PHONY: coverage
coverage:
	go test -v -race -buildvcs -covermode atomic -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: coveralls
coveralls:
	go run github.com/mattn/goveralls@latest -coverprofile=coverage.out -service=github

.PHONY: run-cri
run-cri:
	CGO_ENABLED=1 sudo -E go run ./cmd/anocird --debug

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
