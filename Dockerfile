FROM golang:1.23.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY . .

RUN go mod tidy
RUN go mod vendor
RUN go mod download

RUN go build -o /chat-app ./server/main.go


FROM alpine:3.12.0

COPY --from=builder /chat-app /chat-app

RUN chmod +x /chat-app

EXPOSE 50051

CMD ["/chat-app"]
