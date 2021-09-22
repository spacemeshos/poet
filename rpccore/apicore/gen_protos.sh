#!/bin/sh
# path to googleapis should be passed as the first argument
# e.g. `./genproto.sh $GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis`
grpc_gateway_path=$(go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway)
googleapis_path="$grpc_gateway_path/third_party/googleapis"

# Generate protos.
protoc -I/usr/local/include -I. \
       -I$googleapis_path \
       --go-grpc_out=. \
       --go_out=. \
       apicore.proto

# Generate the REST reverse proxy.
protoc -I/usr/local/include -I. \
       -I$googleapis_path \
       --grpc-gateway_out=logtostderr=true:. \
       apicore.proto

# Generate the swagger file which describes the REST API.
protoc -I/usr/local/include -I. \
       -I$googleapis_path \
       --swagger_out=logtostderr=true:. \
       apicore.proto
