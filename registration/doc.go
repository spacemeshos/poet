/*
Package registration provides functionality for registering PoET challenges, scheduling rounds, and service proofs.

Each round starts by being open for receiving challenges (byte arrays) from miners.
Once finalized (according to the configured duration), a hash digest is created from the received challenges,
and is used as the input for the proof generation. The hash is the root of a merkle tree constructed from registered
challenges sorted lexicographically.
The proof generation is done by a worker, which can be a separate process that communicates with the registration
service.

Once completed, the proof is stored in a database and available for nodes via a GRPC query.
In addition to the PoET, the proof also contains the list of received challenges, so that the membership
of each challenge can be publicly verified.
*/package registration
