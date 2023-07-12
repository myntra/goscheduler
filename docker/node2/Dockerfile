FROM golang:1.17

RUN apt-get update && apt-get install -y ca-certificates openssl

ARG cert_location=/usr/local/share/ca-certificates

RUN openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${cert_location}/github.crt

RUN openssl s_client -showcerts -connect proxy.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM >  ${cert_location}/proxy.golang.crt

RUN update-ca-certificates

WORKDIR /go/src/goscheduler

COPY . .

RUN go env -w GO111MODULE="on"


RUN go get -d -v ./...

ENV INSECURE=1

## Build the binary
RUN go build -o goscheduler .

ENV CASSANDRA_HOST=cassandra

COPY ./docker/node2/start.sh /start.sh
RUN chmod +x /start.sh

ENV hostname="service2"

RUN sed -i -e "s|<hostname>|$hostname|" ./conf/conf.docker.json


EXPOSE 8081

CMD ["/start.sh"]