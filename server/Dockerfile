# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/hekate-server ./cmd/hekate-server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/hekate-server /hekate-server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/hekate-server"]
