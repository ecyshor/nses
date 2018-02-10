FROM golang:1
WORKDIR /go/src/github.com/ecyshor/nses
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["nses"]