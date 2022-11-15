FROM golang:1.19 as build
RUN apt-get update \
   && apt-get install -qy --no-install-recommends \
   unzip

WORKDIR /build/

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN --mount=type=cache,id=build,target=/root/.cache/go-build make build

FROM ubuntu:22.04

COPY --from=build /build/poet /bin/poet
COPY --from=build /build/bin/libgpu-setup.so /lib/

ENTRYPOINT ["/bin/poet"]
