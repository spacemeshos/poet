package hash

import "github.com/spacemeshos/sha256-simd"

func GenMerkleHashFunc(challenge []byte) func(lChild, rChild []byte) []byte {
	return func(lChild, rChild []byte) []byte {
		children := append(lChild, rChild...)
		result := sha256.Sum256(append(challenge, children...))
		return result[:]
	}
}

func GenLabelHashFunc(challenge []byte) func(data []byte) []byte {
	return func(data []byte) []byte {
		result := sha256.Sum256(append(challenge, data...))
		return result[:]
	}
}
