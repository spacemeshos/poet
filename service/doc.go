/*
Package service provides functionality for creating proofs in dedicated rounds, so that
they can be shared among arbitrary number of miners, and thus amortize the CPU cost per proof.

Each round starts by being open for receiving challenges (byte arrays) from miners.
Once finalized (according to the configured duration), a hash digest is created from the received challenges,
and is used as the input for the proof generation.
Once completed, the proof is broadcast to the Spacemesh network via the configured list of gateway nodes.
In addition to the PoET, the proof also contains the list of received challenges, so that the membership
of each challenge can be publicly verified.
*/
package service
