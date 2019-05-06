#!/bin/sh
# path to googleapis should be passed as the first argument
# e.g. `./genproto.sh $GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis`

# Generate protos.
protoc -I/usr/local/include -I. \
       -I$1 \
       --go_out=plugins=grpc:. \
       apicore.proto

# Generate the REST reverse proxy.
protoc -I/usr/local/include -I. \
       -I$1 \
       --grpc-gateway_out=logtostderr=true:. \
       apicore.proto

# Generate the swagger file which describes the REST API.
protoc -I/usr/local/include -I. \
       -I$1 \
       --swagger_out=logtostderr=true:. \
       apicore.proto
