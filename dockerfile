FROM golang:1.25-alpine3.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /configgen ./cmd/configgen

FROM alpine:3.23
RUN apk --no-cache add ca-certificates tzdata
ENV GIN_MODE=release
COPY --from=builder /configgen /usr/local/bin/configgen
EXPOSE 8080
ENTRYPOINT ["configgen"]
