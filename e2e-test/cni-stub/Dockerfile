FROM ubuntu:20.04

RUN  apt-get update \
  && apt-get install -y wget tar build-essential \
  && rm -rf /var/lib/apt/lists/*

# install go
RUN wget https://golang.org/dl/go1.23.0.linux-amd64.tar.gz && rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
ENV PATH "/usr/local/go/bin:${PATH}"
WORKDIR /usr/local/app
COPY . .
# Build
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /usr/local/app/cni cni.go
CMD ["./cni"]