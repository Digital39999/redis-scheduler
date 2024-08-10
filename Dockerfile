FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY ./app ./
WORKDIR /app
RUN go build -o /app/main .

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/main /app/main

CMD ["/app/main"]
