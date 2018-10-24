package internal

import (
	"errors"
	"fmt"
	"github.com/spacemeshos/poet-ref/shared"
	"io/ioutil"
	"path/filepath"
)

type SMProver struct {
	x     []byte   // commitment
	n     uint     // n param 1 <= n <= 63
	h     HashFunc // Hx()
	f     BinaryStringFactory
	phi   shared.Label
	store IKvStore
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
		f: NewSMBinaryStringFactory(),
	}

	dir, err := ioutil.TempDir("", "poet")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Dag temp folder: %s\n", dir)

	store, err := NewKvFileStore(filepath.Join(dir, "store.bin"), n)
	if err != nil {
		return res, err
	}

	err = store.Reset()
	if err != nil {
		return res, err
	}

	res.store = store
	return res, nil
}

func (p *SMProver) DeleteStore() {
	p.store.Delete()
}

// for testing
func (p *SMProver) GetLabel(id Identifier) (shared.Label, bool) {

	inStore, err := p.store.IsLabelInStore(id)
	if err != nil {
		println(err)
		return shared.Label{}, false
	}

	if !inStore {
		println("Warning: label not found in store")
		return shared.Label{}, false
	}

	l, err := p.store.Read(id)
	if err != nil {
		println(err)
		return shared.Label{}, false
	}

	return l, true
}

func (p *SMProver) GetHashFunction() HashFunc {
	return p.h
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

		// if we didn't send the label of this node already then
		// add it to the list
		if _, ok := m[Identifier(id)]; !ok {
			// add the identifier label to the labels list
			label := p.readLabel(Identifier(id))
			labels = append(labels, label)
			m[Identifier(id)] = label
		}

		siblingsIds, err := bs.GetBNSiblings(false)
		if err != nil {
			return Proof{}, err
		}

		for _, siblingId := range siblingsIds { // siblings ids up the path from the leaf to the root
			sibId := siblingId.GetStringValue()
			if _, ok := m[Identifier(sibId)]; !ok {
				// label was not already included in this proof

				// get its value - currently from the memory store
				sibLabel := p.readLabel(Identifier(sibId))

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
	return creteNipChallenge(p.phi, p.h, p.n)
}

func (p *SMProver) GetNonInteractiveProof() (Proof, error) {
	c, err := p.creteNipChallenge()
	if err != nil {
		return Proof{}, err
	}

	return p.GetProof(c)
}

func (p *SMProver) ComputeDag(callback shared.ProofCreatedFunc) {

	rootLabel, err := p.computeDag(shared.RootIdentifier)

	if err != nil {
		callback(shared.Label{}, err)
	}

	p.phi = rootLabel
	//p.printDag("")
	callback(rootLabel, nil)
}

func (p *SMProver) printDag(rootId Identifier) {
	if rootId == "" {
		items := p.store.Size() / shared.WB
		fmt.Printf("DAG: # of nodes: %d. n: %d\n", items, p.n)
	}

	if uint(len(rootId)) < p.n {
		p.printDag(rootId + "0")
		p.printDag(rootId + "1")
	}

	ok, err := p.store.IsLabelInStore(rootId)

	if !ok || err != nil {
		fmt.Printf("Missing label value from map for is %s", rootId)
		return
	}

	label := p.readLabel(rootId)

	fmt.Printf("%s: %s\n", rootId, GetDisplayValue(label))
}

// Compute Dag with a root
func (p *SMProver) computeDag(rootId Identifier) (shared.Label, error) {

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

		leftNodeLabel, err = p.computeDag(leftNodeId)
		if err != nil {
			return shared.Label{}, err
		}

		rightNodeLabel, err = p.computeDag(rightNodId)
		if err != nil {
			return shared.Label{}, err
		}
	}

	// pack data to hash - hx(rootId, leftSibLabel, rightSibLabel)
	labelData := append([]byte(rootId), leftNodeLabel[:]...)
	labelData = append(labelData, rightNodeLabel[:]...)

	// compute root label, store and return it
	labelValue := p.h.Hash(labelData)
	p.writeLabel(rootId, labelValue)
	return labelValue, nil
}

func (p *SMProver) writeLabel(id Identifier, l shared.Label) {
	err := p.store.Write(id, l)
	if err != nil {
		println(err)
		panic(err)
	}
}

func (p *SMProver) readLabel(id Identifier) shared.Label {
	l, err := p.store.Read(id)
	if err != nil {
		println(err)
		panic(err)
	}

	return l
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
		parentValue := p.readLabel(Identifier(parentId.GetStringValue()))
		data = append(data, parentValue[:]...)
	}

	// note that the leftmost leaf has no parents in the dag
	label := p.h.Hash(data)

	// store it
	p.writeLabel(leafId, label)
	return label, nil
}
