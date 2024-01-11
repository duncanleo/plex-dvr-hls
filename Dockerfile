FROM golang:1.21-alpine as app-build

WORKDIR /app
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /bin/app cmd/*.go

FROM collelog/ffmpeg:4.4-alpine-vaapi-amd64

RUN apk add ffmpeg

COPY --from=app-build /bin/app /bin/app
WORKDIR /app

COPY templates/ ./templates/

ENTRYPOINT ["/bin/app"]
