#build
FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY pkg ./pkg
COPY services/sender-worker ./services/sender-worker

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/sender-worker ./services/sender-worker/cmd/sender-worker

#run
FROM gcr.io/distroless/base-debian12
COPY --from=build /out/sender-worker /sender-worker
USER nonroot:nonroot
ENTRYPOINT ["/sender-worker"]
