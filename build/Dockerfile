# Этап компиляции
FROM golang:1.22 AS builder

WORKDIR /app

COPY ../go.mod ../go.sum ./
RUN go mod download

COPY ../ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gopher-equalizer-system ./cmd/main.go

# Этап сборки
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gopher-equalizer-system .

COPY ../config/config.yml ./config/config.yml

COPY internal/database/migrations ./internal/database/migrations

EXPOSE 8080

CMD ["./gopher-equalizer-system"]
