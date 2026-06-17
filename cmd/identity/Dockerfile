FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -o identity ./cmd/identity

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/identity .
COPY scripts/wait-for.sh /usr/local/bin/wait-for.sh
RUN chmod +x /usr/local/bin/wait-for.sh
RUN apk add --no-cache netcat-openbsd curl
EXPOSE 8040
CMD ["/usr/local/bin/wait-for.sh", "postgres:5432", "--", "./identity"]