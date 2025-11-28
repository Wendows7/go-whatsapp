FROM golang:1.25-alpine as builder

WORKDIR /go/src/go-whatsapp

COPY . .

RUN go mod tidy
RUN go build -o app

FROM alpine as runner

WORKDIR /go-whatsapp

COPY --from=builder /go/src/go-whatsapp/app .

CMD ["./app"]

