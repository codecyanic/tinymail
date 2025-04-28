build:
	./build.sh

run: static/source.tgz
	go run .

.PHONY: build run

static/source.tgz:
	git archive HEAD -o static/source.tgz
