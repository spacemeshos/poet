package shared

const (
	T  = 150 // security param
	W  = 256 // bits - label size and Hx() output size
	WB = 32  // W in bytes
)

type HashFunc interface {
	Hash(data []byte) [WB]byte
}

type Label [WB]byte    // label is WB bytes long
type Labels []Label    // an ordered list of Labels
type Identifier string // variable-length binary string. e.g. "0011010" Only 0s and 1s are allows chars. Identifiers are n bits long.

type Challenge struct {
	Data [T]Identifier // A list of T identifiers
}

type Proof struct {
	Phi Label     // dag root label value
	L   [T]Labels // T lists of labels - one for every of the T challenges
}

type IBasicVerifier interface {

	// Verify proof p provided for challenge c using a verifier initialized with x and n where T and W shared between verifier and prover
	Verify(c Challenge, p Proof) bool

	// Create a NIP challenge based on Phi (root label value provided by a proof)
	CreteNipChallenge(phi []byte) (Challenge, error)

	// create a random challenge, that consists of T random identifies (each n bits long)
	CreteRndChallenge() (Challenge, error)
}
