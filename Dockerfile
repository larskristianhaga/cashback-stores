FROM golang:1.23-alpine AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -v -o /run-app .

FROM alpine:3.21

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]
