FROM golang:1.22-alpine as build
RUN apk add libc6-compat gcc musl-dev make
WORKDIR /build/

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
ARG version
RUN --mount=type=cache,id=build,target=/root/.cache/go-build make build VERSION=${version}

FROM alpine
COPY --from=build /build/poet /bin/poet
ENTRYPOINT ["/bin/poet"]
