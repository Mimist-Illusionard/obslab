FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux go build -o /out/redditclone ./cmd/redditclone

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/redditclone /usr/local/bin/redditclone
COPY --from=builder /src/static/ ./static/

USER app

EXPOSE 8080

ENTRYPOINT ["redditclone"]
CMD ["--port", "8080"]