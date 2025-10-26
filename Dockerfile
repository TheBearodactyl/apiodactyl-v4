FROM golang:1.25 AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc sqlite3 libsqlite3-dev && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o apiodactyl ./cmd/apiodactyl/main.go

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    sqlite3 libsqlite3-0 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/apiodactyl .
COPY .env .env
COPY ./files files

EXPOSE 18081

CMD ["./apiodactyl"]
