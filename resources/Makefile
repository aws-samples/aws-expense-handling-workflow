PACKAGES := $(wildcard */.)

default: build

dep:
	go get ./...

build:
	for dir in $(PACKAGES); do \
		mkdir -p $$dir/bin/; \
		GOOS=linux GOARCH=amd64 go build -o $$dir/bin/ $$dir/*.go; \
	done

clean:
	for dir in $(PACKAGES); do \
		rm -rf $$dir/bin/; \
	done

.PHONY: dep clean build
