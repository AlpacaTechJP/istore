//go:generate msgp
package istore

import (
	"encoding/binary"
)

type ItemId uint64

func (id ItemId) Bytes() []byte {
	b := make([]byte, 8, 8)
	binary.PutUvarint(b, uint64(id))
	return b
}

func ToItemId(val []byte) ItemId {
	id, _ := binary.Uvarint(val)
	return ItemId(id)
}

func (id ItemId) Key() []byte {
	return append([]byte(_PathSeqNS), id.Bytes()...)
}

type ItemMeta struct {
	ItemId   ItemId                 `json:"_id,omitempty" msg:"_id,omitempty"`
	FilePath string                 `json:"_filepath,omitempty" msg:"_filepath,omitempty"`
	MetaData map[string]interface{} `json:"metadata,omitempty" msg:"metadata,omitempty"`
}
