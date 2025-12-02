FROM golang:1.22 AS build

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rag-server .

FROM gcr.io/distroless/base-debian12
WORKDIR /srv
COPY --from=build /app/rag-server .
ENV PORT=8082
EXPOSE 8082
ENTRYPOINT ["/srv/rag-server"]
