FROM golang:1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/postigo ./cmd/postigo

FROM alpine:latest
COPY --from=builder /usr/local/bin/postigo /usr/local/bin/postigo

EXPOSE 9090
ENV POSTIGO_SERVER_BIND_PORT=9090
ENV POSTIGO_SERVER_BIND_ADDR=0.0.0.0
CMD [ "/usr/local/bin/postigo", "serve"]