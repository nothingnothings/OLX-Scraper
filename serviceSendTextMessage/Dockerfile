FROM golang:alpine AS servicewa

RUN apk add --no-cache git bash tzdata

WORKDIR /code/serviceSendTextMessage

COPY serviceSendTextMessage/go.mod .
COPY serviceSendTextMessage/go.sum .

RUN go mod download

COPY serviceSendTextMessage .

RUN go build -o /serviceSendTextMessage .

RUN mkdir -p /tmp/WhatsAppSession

CMD ["/serviceSendTextMessage"]