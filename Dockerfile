FROM golang:1.23.4-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=1 CGO_LDFLAGS="-lm" go build -o akeyless-to-bitwarden main.go

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/akeyless-to-bitwarden .

CMD ["./akeyless-to-bitwarden"]
