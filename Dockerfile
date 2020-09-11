FROM golang:1.14
WORKDIR /go/src/ccm/
env http_proxy "http://proxy.ubisoft.org:3128"
env https_proxy "http://proxy.ubisoft.org:3128"
env ftp_proxy "http://proxy.ubisoft.org:3128"
ADD main.go /go/src/ccm/
COPY ./exporter /go/src/ccm/exporter/
COPY ./google.golang.org /go/src/google.golang.org/
COPY ./github.com /go/src/github.com/
COPY ./go.mongodb.org /go/src/go.mongodb.org/
COPY ./golang.org /go/src/golang.org/
EXPOSE 8710
RUN go build -o ccm-tr-exporter .
CMD ["/go/src/ccm/ccm-tr-exporter"]



