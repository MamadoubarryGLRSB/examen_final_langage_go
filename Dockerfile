FROM golang:1.22-alpine AS build

ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o urlwatch ./cmd/urlwatch

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /app/urlwatch .
EXPOSE 8080
ENV STORE=sqlite
ENV SQLITE_PATH=/app/urlwatch.db
ENTRYPOINT ["./urlwatch"]
