FROM denismakogon/ffmpeg-alpine:4.0-buildstage as build-stage
FROM golang:1.21-alpine as app-build

WORKDIR /app
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /bin/app cmd/*.go

FROM alpine:3.15.0 as app

# Copy ffmpeg runtime https://github.com/denismakogon/ffmpeg-alpine#custom-runtime
COPY --from=build-stage /tmp/fakeroot/bin /usr/local/bin
COPY --from=build-stage /tmp/fakeroot/share /usr/local/share
COPY --from=build-stage /tmp/fakeroot/include /usr/local/include
COPY --from=build-stage /tmp/fakeroot/lib /usr/local/lib

COPY --from=app-build /bin/app /bin/app
WORKDIR /app

COPY templates/ ./templates/

ENTRYPOINT ["/bin/app"]

