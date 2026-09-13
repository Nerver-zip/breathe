APP := breath

.PHONY: build run test vet fmt check clean

build:
	go build -o bin/$(APP) .

run:
	go run .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

check: fmt test vet build

clean:
	rm -rf bin coverage.out
