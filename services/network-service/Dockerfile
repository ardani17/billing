FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ ./pkg/
COPY services/network-service/ ./services/network-service/
RUN cd services/network-service && GOWORK=off go build -o /app/server ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 3002
CMD ["./server"]
