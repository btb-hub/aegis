FROM golang:1.22 AS build
WORKDIR /src
COPY devtools/alert-simulator/go.mod devtools/alert-simulator/go.sum* ./
RUN go mod download
COPY devtools/alert-simulator/ ./
RUN CGO_ENABLED=0 go build -o /alert-simulator ./cmd/alert-simulator

FROM gcr.io/distroless/static-debian12
COPY --from=build /alert-simulator /alert-simulator
ENTRYPOINT ["/alert-simulator"]
