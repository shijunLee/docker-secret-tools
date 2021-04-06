# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests and go resource
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY pkg pkg/
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
ARG TAG
ARG BRANCH
ARG BUILD_TIME
ARG COMMIT_ID
RUN GOPROXY=https://goproxy.cn,direct go mod download
# Build
RUN FLAG=`echo "-X github.com/shijunLee/docker-secret-tools/pkg/version.CommitId=${COMMIT_ID} -X github.com/shijunLee/docker-secret-tools/pkg/version.Branch=${BRANCH} -X github.com/shijunLee/docker-secret-tools/pkg/version.Tag=${TAG} -X github.com/shijunLee/docker-secret-tools/pkg/version.BuildTime=${BUILD_TIME}"` && \
    GOPROXY=https://goproxy.io,direct CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build  -ldflags "${FLAG}"  -a -o manager main.go && \
    chmod +X manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:3.12.4
RUN sed -i 's!http://dl-cdn.alpinelinux.org/!https://mirrors.ustc.edu.cn/!g' /etc/apk/repositories && \
    apk --no-cache add tzdata curl bash vim && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates
WORKDIR /
COPY --from=builder /workspace/manager .
ENTRYPOINT ["/manager"]
