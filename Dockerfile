FROM golang:1.24.1 AS buildstage

RUN mkdir /app
ADD . /app
WORKDIR /app


RUN go mod download
RUN go build -o main ./cmd/blindbit-scan/main.go

FROM busybox
COPY --from=buildstage /app/main .

# CA certificates
COPY --from=buildstage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["./main"]
