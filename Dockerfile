FROM golang:1 as build
COPY . .
RUN apt update && apt install unzip
RUN curl -OL https://github.com/google/protobuf/releases/download/v3.4.0/protoc-3.4.0-linux-x86_64.zip && \
    unzip protoc-3.4.0-linux-x86_64.zip -d protoc3 && \
    mv protoc3/bin/* /usr/local/bin/ && \
    mv protoc3/include/* /usr/local/include/ && \
    ln -s /protoc3/bin/protoc /usr/bin/protoc
WORKDIR /go/src/github.com/ecyshor/nses
RUN go get -u github.com/golang/protobuf/protoc-gen-go && protoc --go_out=plugins=grpc:. *.proto
RUN go get -d -v ./...
RUN go install -v ./...
CMD ["nses"]