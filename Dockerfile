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
RUN go build -tags debug
RUN make buildrunner

WORKDIR /go/src/github.com/spacemeshos/poet/cmd/grpc_shutdown
RUN go build

FROM alpine AS spacemesh

RUN apk add --update coreutils

ARG GCLOUD_KEY

ENV GCLOUD_KEY ${GCLOUD_KEY}
ENV POET_EXEC_PATH=/bin/poet

COPY --from=server_builder /go/src/github.com/spacemeshos/poet/poet $POET_EXEC_PATH
COPY --from=server_builder /go/src/github.com/spacemeshos/poet/cmd/runner/runner /bin/runner
COPY --from=server_builder /go/src/github.com/spacemeshos/poet/cmd/grpc_shutdown/grpc_shutdown /bin/grpc_shutdown

RUN echo $GCLOUD_KEY | base64 --decode > spacemesh.json
ENV GOOGLE_APPLICATION_CREDENTIALS=./spacemesh.json

ENTRYPOINT ["/bin/runner"]
