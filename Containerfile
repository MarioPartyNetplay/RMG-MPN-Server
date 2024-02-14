FROM golang:1.20 as builder
WORKDIR /workspace

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -a -o mpn-server .

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
WORKDIR /

COPY --from=builder /workspace/mpn-server .

ENTRYPOINT ["/mpn-server"]
