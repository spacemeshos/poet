// Code generated by github.com/spacemeshos/go-scale/scalegen. DO NOT EDIT.

// nolint
package shared

import (
	"github.com/spacemeshos/go-scale"
)

func (t *InitialPost) EncodeScale(enc *scale.Encoder) (total int, err error) {
	{
		n, err := t.Proof.EncodeScale(enc)
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := t.Metadata.EncodeScale(enc)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (t *InitialPost) DecodeScale(dec *scale.Decoder) (total int, err error) {
	{
		n, err := t.Proof.DecodeScale(dec)
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := t.Metadata.DecodeScale(dec)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (t *Challenge) EncodeScale(enc *scale.Encoder) (total int, err error) {
	{
		n, err := scale.EncodeByteSlice(enc, t.NodeID)
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := scale.EncodeByteSlice(enc, t.PositioningAtxId)
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := scale.EncodeCompact32(enc, uint32(t.PubLayerId))
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := scale.EncodeOption(enc, t.InitialPost)
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := scale.EncodeByteSlice(enc, t.PreviousATXId)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (t *Challenge) DecodeScale(dec *scale.Decoder) (total int, err error) {
	{
		field, n, err := scale.DecodeByteSlice(dec)
		if err != nil {
			return total, err
		}
		total += n
		t.NodeID = field
	}
	{
		field, n, err := scale.DecodeByteSlice(dec)
		if err != nil {
			return total, err
		}
		total += n
		t.PositioningAtxId = field
	}
	{
		field, n, err := scale.DecodeCompact32(dec)
		if err != nil {
			return total, err
		}
		total += n
		t.PubLayerId = LayerID(field)
	}
	{
		field, n, err := scale.DecodeOption[InitialPost](dec)
		if err != nil {
			return total, err
		}
		total += n
		t.InitialPost = field
	}
	{
		field, n, err := scale.DecodeByteSlice(dec)
		if err != nil {
			return total, err
		}
		total += n
		t.PreviousATXId = field
	}
	return total, nil
}