# syntax=docker/dockerfile:1
# Build from repo module root: docker build -f services/common/Dockerfile .
ARG GO_VERSION=1.26-alpine

FROM golang:${GO_VERSION} AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/server ./services/common/cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /out/server /app/server
EXPOSE 8083
CMD ["/app/server"]
