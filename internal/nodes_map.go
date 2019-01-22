package internal

import "github.com/spacemeshos/poet-ref/shared"

// A (k,v) store of (BinaryString, shared.Label)

type NodesMap struct {
	m map[uint]map[uint64]shared.Label
}

func NewNodesMap() *NodesMap {
	return &NodesMap{
		make(map[uint]map[uint64]shared.Label),
	}
}

func (nm *NodesMap) Put(id BinaryString, l shared.Label) {
	d := id.GetDigitsCount()
	v, ok := nm.m[d]
	if !ok {
		v = make(map[uint64]shared.Label)
		nm.m[d] = v
	}
	v[id.GetValue()] = l
}

func (nm *NodesMap) Get(id BinaryString) (shared.Label, bool) {
	v, ok := nm.m[id.GetDigitsCount()]
	if !ok {
		return nil, false
	}
	label, ok := v[id.GetValue()]
	return label, ok
}
