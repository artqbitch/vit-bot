FROM golang:1.26.5-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY base ./base
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/vit-bot .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 bot
WORKDIR /app
COPY --from=build /out/vit-bot /app/vit-bot
USER bot
ENTRYPOINT ["/app/vit-bot"]
