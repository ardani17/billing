FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ ./pkg/
COPY services/billing-api/ ./services/billing-api/
RUN cd services/billing-api && GOWORK=off go build -o /app/server ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY services/billing-api/migrations ./migrations
EXPOSE 3001
CMD ["./server"]
