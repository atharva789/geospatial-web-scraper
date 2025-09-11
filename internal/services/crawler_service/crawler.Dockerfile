# write 2-stage Dockerfile: Builder image, Runner Image (lighter)

FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .

RUN go mod tidy && go build -o crawler_service_out

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/crawler_service_out .
CMD ["./crawler_service_out"]