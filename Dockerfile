FROM golang:1.23.4-alpine3.21 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=1 CGO_LDFLAGS="-lm" go build -o akeyless-to-bitwarden main.go

FROM alpine:3.21
RUN apk --no-cache add ca-certificates libc6-compat
WORKDIR /app
COPY --from=builder /app/akeyless-to-bitwarden .

CMD ["./akeyless-to-bitwarden"]
