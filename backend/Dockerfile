FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/ddz-server ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/ddz-server /bin/ddz-server

EXPOSE 8080

CMD ["/bin/ddz-server"]
