FROM golang:1.19-alpine3.16 as build
RUN apk add libc6-compat gcc musl-dev
WORKDIR /build/

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /build/poet

FROM alpine
COPY --from=build /build/poet /bin/poet
ENTRYPOINT ["/bin/poet"]
