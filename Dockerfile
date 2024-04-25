FROM golang:1.21-alpine3.19 as app-build

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

COPY --from=app-build /bin/app /bin/app
WORKDIR /app

COPY templates/ ./templates/

ENTRYPOINT ["/bin/app"]
