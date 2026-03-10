FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /cocopilot ./cmd/cocopilot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /cocopilot /usr/local/bin/cocopilot
COPY migrations/ /app/migrations/
WORKDIR /app
VOLUME /data
ENV COCO_DB_PATH=/data/tasks.db
ENV COCO_HTTP_ADDR=0.0.0.0:8080
EXPOSE 8080
ENTRYPOINT ["cocopilot"]
