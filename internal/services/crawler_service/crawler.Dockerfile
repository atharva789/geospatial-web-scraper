
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
# RUN ping proxy.golang.org
# Download dependencies
RUN apk add --no-cache git
ENV GOPROXY=direct
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o crawler_service_out ./cmd/main.go

# Stage 2: Create the minimal runner image
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/crawler_service_out .

CMD ["./crawler_service_out"]
