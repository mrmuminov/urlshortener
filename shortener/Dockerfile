# Dockerfile for shortener and redirect
FROM golang:1.19-alpine
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o main .
CMD ["./main"]
