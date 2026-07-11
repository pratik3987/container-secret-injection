FROM golang:1.25 AS builder
WORKDIR /src
COPY . .
RUN go mod tidy
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.0
RUN /go/bin/controller-gen object:headerFile="" paths="./api/..."
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/controller ./cmd/controller

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/controller /usr/local/bin/controller
ENTRYPOINT ["/usr/local/bin/controller"]
