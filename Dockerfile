FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=1 CGO_LDFLAGS="-lm" go build -o akeyless-to-bitwarden main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates libc6-compat
WORKDIR /app
COPY --from=builder /app/akeyless-to-bitwarden .
COPY config.yaml.example ./

CMD ["./akeyless-to-bitwarden"]
