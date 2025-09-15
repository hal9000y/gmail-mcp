FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o gmail-mcp ./cmd/gmail-mcp

FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    pandoc \
    poppler-utils

WORKDIR /app

COPY --from=builder /app/gmail-mcp .

RUN mkdir -p /data

EXPOSE 3000

VOLUME ["/data"]

ENTRYPOINT ["./gmail-mcp"]
CMD ["-stdio", "-http-addr=:3000", \
    "-oauth-token-file=/data/gmail-mcp-token.json", "-oauth-url=http://127.0.0.1:3000/oauth", \
    "-log-file=/data/gmail-mcp.log"]
