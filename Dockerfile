FROM golang:1.17.3-bullseye

WORKDIR /app
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

CMD ["go", "run", "cmd/main.go"]

