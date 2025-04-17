FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git 

WORKDIR /app

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o paste-service .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/paste-service /paste-service

RUN adduser -D -g '' appuser
USER appuser

ENV GIN_MODE=release
ENV SERVER_PORT=8080
ENV GO111MODULE=on

EXPOSE 8080

CMD ["./paste-service"] 