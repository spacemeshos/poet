package shared

const (
	T  = 150 // security param
	W  = 256 // bits
	WB = 32  // w in bytes
)

type HashFunc interface {
	Hash(data []byte) [WB]byte
}

type Label [WB]byte    // label is WB bytes
type Labels []Label    // an ordered list of Labels
type Identifier string // variable length binary string

type Proof struct {
	Phi Label     // dag root label
	L   [T]Labels // a list of T lists of labels
}

type Challenge struct {
	Data [T]Identifier
}

type IBasicVerifier interface {

	// Verify proof p for challenge c with verifier initialized with x and n
	Verify(c Challenge, p Proof) bool

	// Create a NIP challenge based on phi (root label value)
	CreteNipChallenge(phi []byte) (Challenge, error)

	CreteRndChallenge() (Challenge, error)
}
