package shared

import "fmt"

const (
	T  = 150 // security param
	W  = 256 // bits - label size and Hx() output size
	WB = 32  // W in bytes
)

type HashFunc interface {

	// todo: support errors

	// Hash takes arbitrary binary data and returns WB bytes
	Hash(data ...[]byte) []byte
	HashSingle(data []byte) []byte
}

type Label []byte      // label is WB bytes long
type Labels []Label    // an ordered list of Labels
type Identifier string // variable-length binary string. e.g. "0011010" Only 0s and 1s are allows chars. Identifiers are n bits long.

const RootIdentifier = Identifier("")

type Challenge struct {
	Data [T]Identifier // A list of T identifiers
}

func (c *Challenge) Print() {
	for idx, data := range c.Data {
		fmt.Printf("[%d]: %s\n", idx, data)
	}
}

type Proof struct {
	Phi Label     // dag root label value
	L   [T]Labels // T lists of labels - one for every of the T challenges
}

type IBasicVerifier interface {

	// Verify proof p provided for challenge c using a verifier initialized with x and n where T and W shared between verifier and prover
	Verify(c Challenge, p Proof) bool

	VerifyNIP(p Proof) (bool, error)

	// Create a NIP challenge based on Phi (root label value provided by a proof)
	CreateNipChallenge(phi Label) (Challenge, error)

	// Create a random challenge, that consists of T random identifies (each n bits long)
	CreateRndChallenge() (Challenge, error)
}

type ProofCreatedFunc func(phi Label, err error)

// A simple POET prover
type IProver interface {
	ComputeDag() (phi Label, err error)
	GetProof(c Challenge) (Proof, error)
	GetNonInteractiveProof() (Proof, error)

	// for testing
	GetLabel(id Identifier) (Label, bool)
	GetHashFunction() HashFunc
	DeleteStore()
}

type MembershipProof struct {
	Index int
	Root  []byte
	Proof [][]byte
}
