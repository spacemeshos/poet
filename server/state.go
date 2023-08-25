package server

import (
	"path/filepath"

	"github.com/spacemeshos/poet/util"
)

const stateFilename = "state.bin"

type state struct {
	PrivKey []byte
}

func (s *state) save(datadir string) error {
	return util.Persist(filepath.Join(datadir, stateFilename), s)
}

func loadState(datadir string) (*state, error) {
	v := &state{}
	if err := util.Load(filepath.Join(datadir, stateFilename), v); err != nil {
		return nil, err
	}

	return v, nil
}
