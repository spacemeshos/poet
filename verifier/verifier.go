package verifier

import (
	"bytes"
	"errors"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
)

type Challenge = shared.Challenge
type Proof = shared.Proof
type IVerifier = shared.IBasicVerifier
type Identifier = shared.Identifier
type HashFunc = shared.HashFunc
type Label = shared.Label

type SMVerifier struct {
	x []byte   // commitment
	n uint     // n param 1 <= n <= 63
	h HashFunc // Hx()
}

// Verify proof p for challenge c.
// Returns true iff the proof is verified, false otherwise
func (s *SMVerifier) Verify(c Challenge, p Proof) bool {

	// We use k,v memory storage to store label values for identifiers
	// as they are unique in p
	m := internal.NewNodesMap()
	f := internal.NewSMBinaryStringFactory()

	// println("Starting verification...")
	// fmt.Printf("Challenges count: %d\n", len(c.Data))

	// Iterate over each identifier in the challenge and verify the proof for it
	for idx, id := range c.Data {
		// fmt.Printf("Verifying challenge # %d for leaf id: %s\n", idx, id)

		leafNodeId, err := f.NewBinaryString(string(id))
		if err != nil {
			return false
		}

		siblingIds, err := leafNodeId.GetBNSiblings(false)
		if err != nil {
			return false
		}

		// for _, sib := range siblingIds {
		//	fmt.Printf("  Sibling: %s\n", sib.GetStringValue())
		// }

		// a slice of all labels included in proof
		proofLabels := p.L[idx][:]
		labelValue, ok := m.Get(leafNodeId)

		if !ok { // leaf is not in cache
			// first label in the list if the leaf (node id) label - read it and remove it from the slice
			// we use labelValue as the label of the node on the path from the leaf to the root, including both leaf and root
			labelValue, proofLabels = proofLabels[0], proofLabels[1:]
			m.Put(leafNodeId, labelValue)
		} else {
			// label is is in cache - we already verified the Merkle path from it to the root
			// so we can skip to the next id...
			// fmt.Printf("  Skipping - already verified proof for label with id %s\n", id)
			continue
		}

		// fmt.Printf(" Varying Merkle proof for leaf node id %s label: %s \n", id, GetDisplayValue(labelValue))

		// todo: dynamically compute next sibling id to avoid expensive call to GetBNSiblings()

		// flag set to true when we already proved a path
		proved := false

		for _, siblingId := range siblingIds { // siblings ids up the path from the leaf to the root

			// todo: is there a better way to find unique node id than to use the slow getStringValue()?
			// what is a unique hash for (digits, value) ????
			// we only use sibId as a key to the cache map

			var sibValue shared.Label
			var ok bool

			sibValue, ok = m.Get(siblingId)

			// fmt.Printf("Considering sibling: %s....\n", sibId)

			if !ok {
				// label is not the k/v mem store - read it from the proof and store it in the k/v store
				// take label from the head of the slice and remove it from the slice
				sibValue, proofLabels = proofLabels[0], proofLabels[1:]
				m.Put(siblingId, sibValue)
			} else {
				// Optimization
				// we already verified the Merkle proof from this sibling to the root
				// so we can skip the rest of the proof for this leaf
				// fmt.Printf("=>> already verified proof from sibling id %s\n", sibValue)
				proved = true
				break
			}

			// calculate the label of the next node up the path to root, based on its parent nodes labels
			parentNodeId, err := siblingId.TruncateLSB()
			if err != nil {
				return false
			}

			// pack data to hash
			if siblingId.IsEven() {
				// hx(siblingParentNodeId, siblingLabel, currentNodeOnPathValue)
				labelValue = s.h.Hash([]byte(parentNodeId.GetStringValue()), sibValue, labelValue)

			} else {
				// hx(siblingParentNodeId, currentNodeOnPathValue, siblingLabel)
				labelValue = s.h.Hash([]byte(parentNodeId.GetStringValue()), labelValue, sibValue)
			}

			// println("  Computed label value: %s", GetDisplayValue(labelValue))
		}

		// final Merkle proof verification - labelValue should be equal to the root label provided by the proof
		if !proved && bytes.Equal(labelValue[:], p.Phi[:]) == false {
			return false
		}

		//
		// question to research: shouldn't we validate that the leaf label
		// value is what provided by prover?
		//

		// compute the challenge leaf node label (his parents are all the siblings
		// and make sure it matches the label provided by the prover

		data := []byte(leafNodeId.GetStringValue())
		for _, siblingId := range siblingIds {

			if siblingId.IsOdd() {
				// only left siblings - siblings with an identifier that ends with 0 are parents of the leaf
				// so we ignore right siblings on the path to the root
				continue
			}

			if val, ok := m.Get(siblingId); ok {
				data = append(data, val[:]...)
			} else {
				// unexpected error - all left siblings should be in the memory map
				return false
			}
		}
		computedLeafLabelVal := s.h.Hash(data)
		providedLeafLabel, ok := m.Get(leafNodeId)

		if bytes.Equal(computedLeafLabelVal[:], providedLeafLabel[:]) == false {
			return false
		}

		// println("  Challenge verified.")
	}

	// println("All challenges verified.")

	return true
}

func (s *SMVerifier) VerifyNIP(p Proof) (bool, error) {
	// use shared common func
	c, err := internal.CreateNipChallenge(p.Phi, s.h, s.n)
	if err != nil {
		return false, err
	}

	return s.Verify(c, p), nil
}

// γ := (Hx(φ,1),...Hx(φ,t))
func (s *SMVerifier) CreateNipChallenge(phi shared.Label) (Challenge, error) {
	// use shared common func
	return internal.CreateNipChallenge(phi, s.h, s.n)
}

// create a random challenge that can be used to challenge a prover
// that created a proof for shared params (x,t,n,w)
func (s *SMVerifier) CreateRndChallenge() (Challenge, error) {
	var data [shared.T]Identifier
	f := internal.NewSMBinaryStringFactory()

	for i := 0; i < shared.T; i++ {
		b, err := f.NewRandomBinaryString(s.n)
		if err != nil {
			return Challenge{}, err
		}
		data[i] = Identifier(b.GetStringValue())
	}

	c := Challenge{Data: data}
	return c, nil
}

// Create a new verifier for commitment X and param n
func New(x []byte, n uint, h HashFunc) (IVerifier, error) {

	if n < 9 || n > 63 {
		return nil, errors.New("n must be in range (9, 63)")
	}

	res := &SMVerifier{
		x: x,
		n: n,
		h: h,
	}

	return res, nil
}
