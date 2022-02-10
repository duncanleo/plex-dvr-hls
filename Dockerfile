FROM denismakogon/ffmpeg-alpine:4.0-buildstage as build-stage
FROM golang:1.17.6-alpine

# Copy ffmpeg runtime https://github.com/denismakogon/ffmpeg-alpine#custom-runtime
COPY --from=build-stage /tmp/fakeroot/bin /usr/local/bin
COPY --from=build-stage /tmp/fakeroot/share /usr/local/share
COPY --from=build-stage /tmp/fakeroot/include /usr/local/include
COPY --from=build-stage /tmp/fakeroot/lib /usr/local/lib

WORKDIR /app
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

CMD ["go", "run", "cmd/main.go"]

