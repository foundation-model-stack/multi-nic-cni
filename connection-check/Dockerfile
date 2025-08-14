#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache-2.0
#

FROM ubuntu:20.04

RUN  apt-get update \
  && apt-get install -y wget tar build-essential \
  && rm -rf /var/lib/apt/lists/*

# install go
RUN wget https://golang.org/dl/go1.24.1.linux-amd64.tar.gz && rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.1.linux-amd64.tar.gz
ENV PATH "/usr/local/go/bin:${PATH}"
RUN mkdir -p /usr/local/app
WORKDIR /usr/local/app
COPY go.mod go.mod
COPY main.go main.go
COPY iperf3.go iperf3.go
COPY cidr.go cidr.go
RUN go mod tidy && go build -o /usr/local/app/concheck
CMD ["./concheck"]