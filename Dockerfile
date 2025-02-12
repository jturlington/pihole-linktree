FROM golang:1.23.4-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o domain-links

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/domain-links .
# Copy templates directory
COPY --from=builder /app/templates ./templates

# Install curl for healthcheck
RUN apk --no-cache add curl

EXPOSE 8080

# Add healthcheck with DNS configuration
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./domain-links"] 