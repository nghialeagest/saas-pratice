# syntax=docker/dockerfile:1
FROM golang:alpine AS builder

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-w -s" -o /bin/saasbase .

# --- Runtime Stage ---
FROM alpine:3.20

WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata curl

COPY --from=builder /bin/saasbase /app/saasbase

ARG MODULE=all
ENV MODULE=${MODULE}

EXPOSE 8080 8081 8082 8083 8084 9090 9091 9092 9093 9094

ENTRYPOINT ["/app/saasbase"]
