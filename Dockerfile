FROM golang:1.16 AS build
ENV GOPATH /go
COPY . /go/src/snekfarm
WORKDIR /go/src/snekfarm
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -o /go/bin/snekfarm
RUN test -e /go/bin/snekfarm

FROM scratch
COPY --from=build /go/bin/snekfarm /go/bin/snekfarm
ENV TZ UTC
EXPOSE 3000/tcp
ENTRYPOINT ["/go/bin/snekfarm"]
