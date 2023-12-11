FROM golang:alpine as builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./
RUN go build -v -o sniproxy

FROM alpine
COPY --from=builder /app/sniproxy /bin/sniproxy
RUN apk add --no-cache ca-certificates

ENTRYPOINT ["/bin/sniproxy"]
