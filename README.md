# poet-ref

This is a private repo designed to create a refernce poet implementatkon for testing of the open source poet server.

The goal is to dsitribute a binary 'blac box' go component of a verifier that can be used to verify proofs created by a prover.

## Build

```
go get -u github.com/kardianos/govendor
govendor sync
go build
```

## Run the tests
```
go test ./...
```
