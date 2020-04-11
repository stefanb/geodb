FROM golang:1.12.9-alpine3.10 as build-env

RUN apk add git
RUN mkdir /geodb
RUN apk --update add ca-certificates
WORKDIR /geodb
COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/geodb
FROM scratch
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build-env /go/bin/geodb /go/bin/geodb
WORKDIR /geodb
ENTRYPOINT ["/go/bin/geodb"]