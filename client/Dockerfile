FROM golang:1.15.3-alpine3.12 as builder

WORKDIR /build
COPY go.mod .
COPY go.sum .
COPY main.go .

RUN CGO_ENABLED=0 go build -o book-client main.go
FROM alpine
COPY --from=builder /build/book-client .
ENTRYPOINT ["./book-client"]