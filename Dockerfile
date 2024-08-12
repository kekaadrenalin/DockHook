FROM --platform=$BUILDPLATFORM golang:1.22.6-alpine AS builder

RUN apk add --no-cache ca-certificates && mkdir /dockhook

WORKDIR /dockhook

# Copy go mod files
COPY go.* ./
RUN go mod download

# Copy all other files
COPY pkg ./pkg
COPY main.go ./

# Args
ARG TAG=dev
ARG TARGETOS TARGETARCH

# Build binary
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$TAG"  -o dockhook

RUN mkdir /data

FROM scratch

LABEL org.opencontainers.image.source="https://github.com/kekaadrenalin/dockhook"
LABEL org.opencontainers.image.title="DockHook Docker Image"
LABEL org.opencontainers.image.licenses="AGPL-3.0"

ENV PATH /bin
COPY --from=builder /data /data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dockhook/dockhook /dockhook

EXPOSE 8080

ENTRYPOINT ["/dockhook"]
