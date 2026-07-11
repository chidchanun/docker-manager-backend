FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
FROM alpine:3.22
COPY --from=build /out/api /usr/local/bin/docker-manager-api
EXPOSE 10002
ENTRYPOINT ["docker-manager-api"]
