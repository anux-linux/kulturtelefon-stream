FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

RUN go mod tidy

COPY . .

RUN go build -C /app/ -o kulturtelefon

FROM debian:stable-slim

WORKDIR /app

COPY --from=builder /app/kulturtelefon .

ENV STREAM_API_PORT=8080

EXPOSE 8080

# CMD ["./kulturtelefon", "--config", "/app/config.yaml", "--loglevel", "info"]
CMD ["./kulturtelefon --config", "${CONFIG_PATH}", "--loglevel", "${LOG_LEVEL}"]
