FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o lvc-coordinator .

FROM scratch
COPY --from=builder /build/lvc-coordinator /usr/local/bin/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
ENV GIN_MODE=release
ENTRYPOINT ["lvc-coordinator"]
