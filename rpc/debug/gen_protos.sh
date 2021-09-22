#!/bin/sh

grpc_gateway_path=$(go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway)
googleapis_path="$grpc_gateway_path/third_party/googleapis"


# Generate protos.
protoc -I. -I$googleapis_path --go-grpc_out=. --go_out=. debug.proto

# Generate web proxy.
protoc -I. -I$googleapis_path --grpc-gateway_out=logtostderr=true:. debug.proto

# Generate the swagger file which describes the REST API.
protoc -I. -I$googleapis_path --swagger_out=logtostderr=true:. debug.proto
