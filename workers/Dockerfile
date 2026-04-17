# syntax=docker/dockerfile:1
# Build from repo module root: docker build -f workers/Dockerfile .
ARG GO_VERSION=1.26-alpine

FROM golang:${GO_VERSION} AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/worker ./workers/cmd/worker

FROM golang:${GO_VERSION} AS dev
WORKDIR /src
RUN apk add --no-cache ca-certificates git curl && go install github.com/air-verse/air@latest
ENV PATH="/root/go/bin:${PATH}"

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/worker /app/worker
ENTRYPOINT ["/app/worker"]
CMD ["--mode=all"]
