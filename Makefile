# Makefile for releasing
#
# The release version is controlled from pkg/version

TAG?=latest
NAME:=shortener
DOCKER_REPOSITORY:=ut8ia
DOCKER_IMAGE_NAME:=$(DOCKER_REPOSITORY)/$(NAME)
GIT_COMMIT:=$(shell git describe --dirty --always)
VERSION:=$(shell grep 'VERSION' pkg/version/version.go | awk '{ print $$4 }' | tr -d '"')
EXTRA_RUN_ARGS?=

run:
	go run cmd/app/*
test:
	go test -v -race ./...
build:
	CGO_ENABLED=0 go build -a -o ./bin/app  ./cmd/app/*
image_version:
	docker build -t $(DOCKER_IMAGE_NAME):$(VERSION) .
image:
	docker build -t $(DOCKER_IMAGE_NAME).
fmt:
	gofmt -l -s -w ./
	goimports -l -w ./

