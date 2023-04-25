FROM --platform=linux/amd64 ubuntu:20.04

RUN  apt-get update \
  && apt-get install -y wget tar build-essential net-tools iproute2 git \
  && rm -rf /var/lib/apt/lists/*

# install go
RUN wget https://golang.org/dl/go1.17.linux-amd64.tar.gz && rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
ENV PATH "/usr/local/go/bin:${PATH}"
RUN mkdir -p /usr/local/app
RUN mkdir -p /host/opt/cni/bin
WORKDIR /usr/local/app
COPY src src
COPY cni cni
COPY run.sh run.sh
RUN cd cni && go mod tidy && go build -o /usr/local/app/multi-nic ./plugins/main/multi-nic
RUN cd cni && go mod tidy && go build -o /usr/local/app/multi-nic-ipam ./plugins/ipam/multi-nic-ipam
RUN cd src && go mod tidy && go build -o /usr/local/app/daemon
RUN rm -r /usr/local/app/src && rm -r /usr/local/app/cni
CMD ["/bin/bash", "run.sh"]