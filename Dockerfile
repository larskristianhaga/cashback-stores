FROM golang:1.24-alpine AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 go build -v -o /run-app .

FROM gcr.io/distroless/static-debian12

EXPOSE 8080

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]
