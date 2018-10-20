package shared

// An immutable variable length binary string with possible leading 0s
type BinaryString interface {

	// Gets string representation. e.g. "00011"
	GetStringValue() string

	// Gets the binary value encoded in the string. e.g. 12
	GetValue() uint64

	// Returns number of digits including leading 0s if any
	GetDigitsCount() uint

	// Returns a new BinaryString with the LSB truncated. e.g. "0110" -> "011"
	TruncateLSB() (BinaryString, error)

	// Returns a new BinaryString with the LSB flipped. e.g. "0110" -> "0111"
	FlipLSB() (BinaryString, error)

	// Returns the siblings on the path from a node identified by the binary string to the root in a full binary tree
	GetBNSiblings() ([]BinaryString, error)
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
