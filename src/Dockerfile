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
COPY startup.sh .

RUN apt-get update && apt-get install -y curl
RUN chmod +x kulturtelefon
RUN chmod +x startup.sh

ENV STREAM_API_PORT=8080

EXPOSE 8080

# Use ENTRYPOINT for the executable
ENTRYPOINT ["./startup.sh"]
