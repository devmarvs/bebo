# syntax=docker/dockerfile:1
ARG GO_VERSION=1.25
FROM golang:${GO_VERSION} AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

ARG APP_PATH=./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/app ${APP_PATH}

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/app /app/app
ENV BEBO_ADDRESS=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/app"]
