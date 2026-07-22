FROM golang:1.23-alpine3.19 AS app-build

WORKDIR /app
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /bin/app cmd/*.go

FROM alpine:3.19

ARG TARGETPLATFORM

RUN case ${TARGETPLATFORM:-linux/amd64} in \
    "linux/amd64") apk add ffmpeg intel-media-driver ;; \
    *)             apk add ffmpeg ;; \
    esac

WORKDIR /app

COPY --from=app-build /bin/app /bin/app

ENTRYPOINT ["/bin/app"]
