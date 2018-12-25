FROM golang:1.11.4-alpine3.8 as build-env

RUN mkdir /upbound
WORKDIR /upbound

RUN apk add --update --no-cache git

# Cache modules where possible
COPY go.mod .
RUN go mod download

COPY main.go .
COPY pkg ./pkg

RUN CGO_ENABLED=0 go build -o /go/bin/upbound

# <- Second step to build minimal image
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/bin/upbound /go/bin/upbound
ENTRYPOINT ["/go/bin/upbound"]