FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/booking-system ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/booking-system .
COPY --from=builder /app/static ./static

EXPOSE 8080

CMD ["./booking-system"]
