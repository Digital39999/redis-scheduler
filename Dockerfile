FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY ./src/main.go ./

RUN go mod download
COPY . .
RUN go build -o /app/main .

FROM alpine:latest

COPY --from=builder /app/main /app/main
CMD ["/app/main"]
