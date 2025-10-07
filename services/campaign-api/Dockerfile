# build
FROM golang:1.23-alpine AS build
WORKDIR /src


COPY go.mod go.sum ./
RUN go mod download


COPY docs ./docs
COPY internal ./internal
COPY pkg ./pkg
COPY services/campaign-api ./services/campaign-api


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/campaign-api ./services/campaign-api/cmd/campaign-api

# run
FROM gcr.io/distroless/base-debian12
WORKDIR /

EXPOSE 8080
COPY --from=build /out/campaign-api /campaign-api

USER nonroot:nonroot
ENTRYPOINT ["/campaign-api"]
