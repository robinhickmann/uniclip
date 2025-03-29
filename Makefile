build:
	go build .

lint:
	docker run -e RUN_LOCAL=true -v "$(shell pwd)":/tmp/lint ghcr.io/super-linter/super-linter:latest

release:
	git tag $(tag)
	git push origin $(tag)
	goreleaser release
