FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mockmcp ./hack/mockmcp

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/mockmcp /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mockmcp"]
CMD ["-addr=:8080"]
