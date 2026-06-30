FROM golang:1.20 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/controller ./cmd/controller

FROM gcr.io/distroless/base-debian11
COPY --from=builder /out/controller /usr/local/bin/controller
ENTRYPOINT ["/usr/local/bin/controller"]
