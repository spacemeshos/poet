package internal

import (
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
)


type SMProver struct {
	x   []byte                      // commitment
	n   uint                        // n param 1 <= n <= 63
	h   HashFunc                    // Hx()
	m   map[Identifier]shared.Label // label store - in memory for now
	f   BinaryStringFactory
	phi shared.Label
}

// Create a new prover with commitment X and param 1 <= n <= 63
func NewProver(x []byte, n uint) (shared.IProver, error) {

	if n < 1 || n > 63 {
		return nil, errors.New("n must be in range [1, 63]")
	}

	res := &SMProver{
		x: x,
		n: n,
		h: shared.NewHashFunc(x),
		m: make(map[Identifier]shared.Label),
		f: NewSMBinaryStringFactory(),
	}

	return res, nil
}

// generate proof and return it
func (p *SMProver) GetProof(c Challenge) (Proof, error) {

	if len(c.Data) != shared.T {
		return Proof{}, errors.New("invalid challenge data")
	}

	proof := Proof{}
	proof.Phi = p.phi
	proof.L = [shared.T]shared.Labels{}

	// temp store use to ensure labels in proof are unique and not duplicated
	var m = make(map[Identifier]shared.Label)

	// Iterate over each identifier in the challenge and create the proof for it
	for idx, id := range c.Data {

		var labels shared.Labels

		bs, err := p.f.NewBinaryString(string(id))
		if err != nil {
			return Proof{}, err
		}

		// add the identifier label to the labels list
		labels = append(labels, p.m[Identifier(id)])

		siblingsIds, err := bs.GetBNSiblings(false)
		if err != nil {
			return Proof{}, err
		}

		for _, siblingId := range siblingsIds { // siblings ids up the path from the leaf to the root
			sibId := siblingId.GetStringValue()
			if _, ok := m[Identifier(sibId)]; !ok {
				// label was not already included in this proof

				// get its value - currently from the memory store
				sibLabel := p.m[Identifier(sibId)]

				// store it in m so we won't add it again in another labels list in the proof
				m[Identifier(sibId)] = sibLabel

				// add it to the list of labels in the proof for identifier id
				labels = append(labels, sibLabel)
			}
		}

		proof.L[idx] = labels
	}

	return proof, nil
}

// γ := (Hx(φ,1),...Hx(φ,t))
func (p *SMProver) creteNipChallenge() (Challenge, error) {
	// use shared common func
	return creteNipChallenge(p.phi[:], p.h, p.n)
}

func (p *SMProver) GetNonInteractiveProof() (Proof, error) {
	c, err := p.creteNipChallenge()
	if err != nil {
		return Proof{}, err
	}

	return p.GetProof(c)
}

func (p *SMProver) ComputeDag(callback shared.ProofCreatedFunc) {

	rootLabel, err := p.computeDagInMemory(shared.RootIdentifier)

	if err != nil {
		callback(shared.Label{}, err)
	}

	p.phi = rootLabel
	println("Map size: ", len(p.m))
	callback(rootLabel, nil)
}

// Compute Dag with a root
func (p *SMProver) computeDagInMemory(rootId Identifier) (shared.Label, error) {

	leftNodeId := rootId + "0"
	rightNodId := rootId + "1"

	childrenHeight := uint(len(rootId)) + 1
	var leftNodeLabel, rightNodeLabel shared.Label
	var err error

	if childrenHeight == p.n { // children are leaves

		leftNodeLabel, err = p.computeLeafLabel(leftNodeId)
		if err != nil {
			return shared.Label{}, err
		}
		rightNodeLabel, err = p.computeLeafLabel(rightNodId)
		if err != nil {
			return shared.Label{}, err
		}

	} else { // children are internal dag nodes

		leftNodeLabel, err = p.computeDagInMemory(leftNodeId)
		if err != nil {
			return shared.Label{}, err
		}

		rightNodeLabel, err = p.computeDagInMemory(rightNodId)
		if err != nil {
			return shared.Label{}, err
		}
	}

	// pack data to hash - hx(rootId, leftSibLabel, rightSibLabel)
	labelData := append([]byte(rootId), leftNodeLabel[:]...)
	labelData = append(labelData, rightNodeLabel[:]...)

	// compute root label, store and return it
	labelValue := p.h.Hash(labelData)
	p.m[rootId] = labelValue
	return labelValue, nil
}

// Given a leaf node with id leafId - return the value of its label
// Pre-condition: all parent label values have been computed and are available for the implementation
func (p *SMProver) computeLeafLabel(leafId Identifier) (shared.Label, error) {

	bs, err := p.f.NewBinaryString(string(leafId))
	if err != nil {
		return shared.Label{}, err
	}

	// generate packed data to hash
	data := []byte(leafId)

	parentIds, err := bs.GetBNSiblings(true)
	if err != nil {
		return shared.Label{}, err
	}

	for _, parentId := range parentIds {
		parentValue := p.m[Identifier(parentId.GetStringValue())]
		data = append(data, parentValue[:]...)
	}

	// note that the leftmost leaf has no parents in the dag
	label := p.h.Hash(data)

	// store it
	p.m[leafId] = label
	return label, nil
}
