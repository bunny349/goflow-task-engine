FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /taskqueue-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /taskqueue-server /usr/local/bin/taskqueue-server

EXPOSE 8080
ENTRYPOINT ["taskqueue-server"]
