FROM golang:1.26.2-bookworm
ENV GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor
WORKDIR /workspace
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
RUN go build -mod=vendor -o /usr/local/bin/signalroute ./cmd/signalroute
CMD ["/usr/local/bin/signalroute", "-addr", "0.0.0.0:21211", "-data", "/tmp/signalroute-data"]
