FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/zaplink ./cmd/api/main.go

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/zaplink /usr/local/bin/zaplink

EXPOSE 8080

ENTRYPOINT ["zaplink"]
