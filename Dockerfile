FROM golang:1.15-alpine as builder

ARG REVISION

RUN mkdir -p /app/

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -ldflags "-s -w \
    -X github.com/ut8ia/shortener/pkg/version.REVISION=${REVISION}" \
    -a -o bin/app cmd/app/*

FROM alpine:3.12

ARG BUILD_DATE
ARG VERSION
ARG REVISION

LABEL maintainer="eugene.anufriev"

RUN addgroup -S app \
    && adduser -S -g app app \
    && apk --no-cache add \
    ca-certificates curl netcat-openbsd

WORKDIR /home/app

COPY --from=builder /app/bin/app .
COPY ./ui ./ui
RUN chown -R app:app ./

USER app

CMD ["./app"]
