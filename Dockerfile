FROM golang:1.16-alpine3.13 AS build_base
RUN apk add bash make git curl unzip rsync libc6-compat gcc musl-dev
WORKDIR /go/src/github.com/spacemeshos/poet

# Force the go compiler to use modules
ENV GO111MODULE=on

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download

# This image builds th
FROM build_base AS server_builder
# Here we copy the rest of the source code
COPY . .

# And compile the project
RUN go build
RUN make buildrunner

WORKDIR /go/src/github.com/spacemeshos/poet/cmd/grpc_shutdown
RUN go build

FROM alpine AS spacemesh
COPY --from=server_builder /go/src/github.com/spacemeshos/poet/poet /bin/poet
COPY --from=server_builder /go/src/github.com/spacemeshos/poet/cmd/runner/runner /bin/runner
COPY --from=server_builder /go/src/github.com/spacemeshos/poet/cmd/grpc_shutdown/grpc_shutdown /bin/grpc_shutdown
ENV EXEC_PATH=/bin/poet
ENTRYPOINT ["/bin/runner"]
