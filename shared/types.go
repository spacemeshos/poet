package shared

// An immutable variable length binary string with possible leading 0s
// e.g.
type BinaryString interface {

	// Get string representation. e.g. "00011"
	GetStringValue() string

	// Get the binary value encoded in the string. e.g. 12
	GetValue() uint64

	// return number of digits including leading 0s if any
	GetDigitsCount() uint

	TruncateLSB() (BinaryString, error)

	FlipLSB() (BinaryString, error)
}

type BinaryStringFactory interface {
	NewBinaryString(s string) (BinaryString, error)
	NewRandomBinaryString(d uint) (BinaryString, error)
	NewBinaryStringFromInt(v uint64, d uint) (BinaryString, error)
}

type HashFunc interface {
	Hash(data []byte) [WB]byte
}

const (
	T  = 150 // security param
	W  = 256 // bits
	WB = 32  // w in bytes
)

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
