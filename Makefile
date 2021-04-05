VERSION ?= $(shell cat ./pkg/version/version.go|grep "version ="|awk '{print $$3}'| sed 's/\"//g' | tr  "\n" " " | tr "\n" " " | sed 's/[[:space:]]//g' | tr "\n" " ")
TAG ?= $(shell git describe --abbrev=0 --tags)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_ID = $(shell git rev-parse --short HEAD)
BUILD_TIME= $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
FLAGS="-X github.com/shijunLee/docker-secret-tools/pkg/version.CommitId=$(COMMIT_ID) -X github.com/shijunLee/docker-secret-tools/pkg/version.Branch=$(BRANCH) -X github.com/shijunLee/docker-secret-tools/pkg/version.Tag=$(TAG) -X github.com/shijunLee/docker-secret-tools/pkg/version.BuildTime=$(BUILD_TIME)"
IMG ?= docker.shijunlee.local/library/docker-secret-tool:$(VERSION)

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...

.PHONY: manager
manager: fmt vet
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build  -ldflags $(FLAGS) -o bin/manager main.go && chmod +x bin/manager

.PHONY: docker-build
docker-build:
	docker build --build-arg TAG="$(TAG)" \
    --build-arg BRANCH="$(BRANCH)" \
    --build-arg COMMIT_ID="$(COMMIT_ID)" \
    --build-arg BUILD_TIME="$(BUILD_TIME)" -t ${IMG} .
	docker images |grep "<none>" |awk '{print $$3}' | xargs docker rmi -f

.PHONY: docker-push
docker-push:
	docker push ${IMG}

docker-all: docker-build docker-push
