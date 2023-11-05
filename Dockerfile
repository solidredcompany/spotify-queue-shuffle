FROM golang:1.21

WORKDIR /app

COPY go.mod ./

COPY *.go ./
COPY internal ./internal
COPY web ./web

RUN CGO_ENABLED=0 GOOS=linux go build -o /queue-shuffle

CMD ["/queue-shuffle"]
