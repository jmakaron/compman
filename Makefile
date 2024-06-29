NAME = compman
TAG  = latest
ifeq ($(VERSION),)
VERSION = $(shell git tag | sort -V | tail -1)
endif
LDFLAGS = "-X main.build=$(shell date -u +%Y-%m%dT%H-%M-%SZ) -X main.version=${VERSION}"
.DEFAULT_GOAL := $(NAME)

.PHONY: docker
docker:
	DOCKER_DEFAULT_PLATFORM=linux/amd64 docker build	\
	-t $(NAME):$(VERSION)								\
	-f cloudbuild/Dockerfile --build-arg VERSION=$(VERSION) .

.PHONY: all
all: clean api ${NAME}

.PHONY: api
api:
	make -C api

.PHONY: ${NAME}
${NAME}:
		mkdir -p bin
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags $(LDFLAGS) -o bin ./...

.PHONY: clean
clean:
		go clean
		rm -rf bin/*		