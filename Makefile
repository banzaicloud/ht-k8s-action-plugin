.DEFAULT_GOAL := help
.PHONY: help build

OS := $(shell uname -s)

GITREV = $(shell git rev-parse --short HEAD)

build: ## builds binary package in ./bin
	go build -o bin/ht-k8s-action-plugin .

clean: ## deletes ./bin directory 
	rm -rf ./bin

docker-build: ## builds a docker image with the binary
	docker build -t banzaicloud/ht-k8s-action-plugin .

deps: ## downloads dep tool if not present
	which dep > /dev/null || go get -u github.com/golang/dep/cmd/dep 

list: ## lists make targets
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

help: ## displays this help message
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

