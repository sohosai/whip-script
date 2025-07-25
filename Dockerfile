FROM golang:1.24 AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o streamer -ldflags="-w -s" main.go

FROM debian:12-slim
WORKDIR /app
COPY --from=build /app/streamer /app/streamer
RUN apt update -y && apt-get install -y ca-certificates

CMD ["/app/streamer"]