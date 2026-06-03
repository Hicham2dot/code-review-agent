FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o code-review-agent ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates git
RUN addgroup -g 1000 reviewer && adduser -D -u 1000 -G reviewer reviewer
WORKDIR /app
COPY --from=builder /app/code-review-agent .
RUN chown -R reviewer:reviewer /app && \
    mkdir -p /app/diffs /app/results && \
    chmod 777 /app/diffs /app/results && \
    mkdir -p /data && \
    chown reviewer:reviewer /data
USER reviewer
EXPOSE 8080
ENTRYPOINT ["./code-review-agent"]
CMD ["server", "--port=8080", "--db=/data/reviews.db"]
