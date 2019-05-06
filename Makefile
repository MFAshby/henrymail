APP := henrymail

default: build lint test

generate:
	@ go generate ./...

build: generate
	@ go build -o dist/$(APP)

clean:
	@ rm -rf dist/ models/

test: build
	go test -cover -race -count=1 ./...

lint:
	@ golint -set_exit_status