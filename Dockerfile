# Start from a Debian-based Golang 1.16 image
FROM golang:1.20-buster

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /ratelimiter

# Start the binary.
CMD [ "/ratelimiter" ]
