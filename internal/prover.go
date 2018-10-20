package internal

import (
	"errors"
	"github.com/spacemeshos/poet-ref/shared"
)

// A simple poet prover

type IProver interface {
	CreateProof(callback ProofCreatedFunc)
}

type ProofCreatedFunc func(phi Label, err error) ()

type Label shared.Label

type SMProver struct {
	x []byte   // commitment
	n uint     // n param 1 <= n <= 63
	h HashFunc // Hx()
	m map[Identifier]shared.Label // label storage - in memory for now
	f BinaryStringFactory

}

// Create a new prover with commitment X and param 1 <= n <= 63
func NewProver(x []byte, n uint) (IProver, error) {

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

func (p* SMProver) CreateProof(callback ProofCreatedFunc) {
	rootLabel, err := p.ComputeDag(Identifier(""))
	if err != nil {
		callback(Label{}, err)
	}

	callback(rootLabel, nil)
}

// Compute Dag with a root
func (p* SMProver) ComputeDag(rootId Identifier) (Label, error) {

	leftNodeId := rootId + "0"
	rightNodId := rootId + "1"

	height := uint(len(rootId)) + 1
	var leftNodeLabel, rightNodeLabel Label
	var err error

	if height == p.n {
		// we are at a leaf
		leftNodeLabel, err = p.computeLeafLabel(leftNodeId)
		if err != nil {
			return Label{}, err
		}
		rightNodeLabel, err = p.computeLeafLabel(rightNodId)
		if err != nil {
			return Label{}, err
		}
	}

	leftNodeLabel, err = p.ComputeDag(leftNodeId)
	if err != nil {
		return Label{}, err
	}
	rightNodeLabel, err = p.ComputeDag(rightNodId)
	if err != nil {
		return Label{}, err
	}

	// compute root label, store it and return it

	// pack data to hash - hx(rootId, leftSibLabel, rightSibLabel)
	labelData := append([]byte(rootId), leftNodeLabel[:]...)
	labelData = append(labelData, rightNodeLabel[:]...)
	labelValue := p.h.Hash(labelData)
	p.m[rootId] = labelValue
	return labelValue, nil
}

// Given a leaf node with id leafId - return the value of its label
// Pre-condition: all parent label values have been computed and are available for the implementation
func (p* SMProver) computeLeafLabel(leafId Identifier) (Label, error) {

	bs, err := p.f.NewBinaryString(string(leafId))
	if err != nil {
		return Label{}, err
	}

	// generate packed data to hash
	data := []byte(leafId)

	parentIds, err := bs.GetBNSiblings(true)
	if err != nil {
		return Label{}, err
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