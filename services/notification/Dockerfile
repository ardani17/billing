FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ ./pkg/
COPY services/notification/ ./services/notification/
RUN cd services/notification && GOWORK=off go build -o /app/server ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 3003
CMD ["./server"]
