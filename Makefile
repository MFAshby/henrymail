APP := henrymail

default: build lint test

generate:
	@ go generate ./...

build: generate
	@ go build -o $(APP)

clean:
	@ rm -f $(APP)

test: build
	go test -cover -race -count=1 ./...

lint:
	@ golint -set_exit_status