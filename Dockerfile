FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY ./app /app
RUN go build -o /app/main .

CMD ["/app/main"]