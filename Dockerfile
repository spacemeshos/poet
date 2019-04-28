FROM golang:1.11.2-alpine3.8 AS build_base
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

FROM alpine AS spacemesh
COPY --from=server_builder /go/src/github.com/spacemeshos/poet/poet /bin/poet
ENTRYPOINT ["/bin/poet"]
