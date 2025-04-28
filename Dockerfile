FROM golang:1.22

RUN go install github.com/evanw/esbuild/cmd/esbuild@v0.25.3

WORKDIR /tinymail
COPY go.mod go.sum ./
RUN go mod download

COPY static static
RUN esbuild static/app.js --bundle --minify \
    --target=es2020,chrome83,firefox76,safari13 \
    --outdir=static --allow-overwrite
RUN esbuild static/style.css --bundle --minify \
    --target=es2020,chrome83,firefox76,safari13 \
    --outdir=static --allow-overwrite

COPY main.go .
COPY internal internal
RUN CGO_ENABLED=0 GOOS=linux go build .
