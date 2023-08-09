# Copyright (c) 2023 Myntra Designs Private Limited.
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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