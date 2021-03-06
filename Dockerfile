######## Start a builder stage #######
FROM golang:1.16-alpine as builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/main cmd/nginx-conf-generator/main.go

######## Start a new stage from scratch #######
FROM alpine:latest

WORKDIR /opt/
COPY --from=builder /app/bin/main .

CMD ["./main"]