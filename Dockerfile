FROM golang:alpine3.15 as build
RUN apk add libc6-compat gcc musl-dev
WORKDIR /build/
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /build/poet

FROM alpine
COPY --from=build /build/poet /bin/poet
ENTRYPOINT ["/bin/poet"]
