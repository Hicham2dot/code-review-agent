FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o code-review-agent ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/code-review-agent .
ENTRYPOINT ["./code-review-agent"]
