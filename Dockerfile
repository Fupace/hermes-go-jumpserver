# Multi-stage build for Hermes JumpServer
FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev sqlite-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o /jumpserver -ldflags="-s -w" .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

COPY --from=builder /jumpserver /usr/local/bin/jumpserver

RUN mkdir -p /data
VOLUME /data

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/jumpserver"]
CMD ["--addr", ":8080", "--data-dir", "/data"]
