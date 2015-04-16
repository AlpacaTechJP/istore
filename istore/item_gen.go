package istore

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *ItemId) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var tmp uint64
		tmp, err = dc.ReadUint64()
		(*z) = ItemId(tmp)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z ItemId) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteUint64(uint64(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z ItemId) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendUint64(o, uint64(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ItemId) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var tmp uint64
		tmp, bts, err = msgp.ReadUint64Bytes(bts)
		(*z) = ItemId(tmp)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

func (z ItemId) Msgsize() (s int) {
	s = msgp.Uint64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *ItemMeta) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "_id":
			{
				var tmp uint64
				tmp, err = dc.ReadUint64()
				z.ItemId = ItemId(tmp)
			}
			if err != nil {
				return
			}
		case "_filepath":
			z.FilePath, err = dc.ReadString()
			if err != nil {
				return
			}
		case "metadata":
			var msz uint32
			msz, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.MetaData == nil && msz > 0 {
				z.MetaData = make(map[string]interface{}, msz)
			} else if len(z.MetaData) > 0 {
				for key, _ := range z.MetaData {
					delete(z.MetaData, key)
				}
			}
			for msz > 0 {
				msz--
				var xvk string
				var bzg interface{}
				xvk, err = dc.ReadString()
				if err != nil {
					return
				}
				bzg, err = dc.ReadIntf()
				if err != nil {
					return
				}
				z.MetaData[xvk] = bzg
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *ItemMeta) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteMapHeader(3)
	if err != nil {
		return
	}
	err = en.WriteString("_id")
	if err != nil {
		return
	}
	err = en.WriteUint64(uint64(z.ItemId))
	if err != nil {
		return
	}
	err = en.WriteString("_filepath")
	if err != nil {
		return
	}
	err = en.WriteString(z.FilePath)
	if err != nil {
		return
	}
	err = en.WriteString("metadata")
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.MetaData)))
	if err != nil {
		return
	}
	for xvk, bzg := range z.MetaData {
		err = en.WriteString(xvk)
		if err != nil {
			return
		}
		err = en.WriteIntf(bzg)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *ItemMeta) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendMapHeader(o, 3)
	o = msgp.AppendString(o, "_id")
	o = msgp.AppendUint64(o, uint64(z.ItemId))
	o = msgp.AppendString(o, "_filepath")
	o = msgp.AppendString(o, z.FilePath)
	o = msgp.AppendString(o, "metadata")
	o = msgp.AppendMapHeader(o, uint32(len(z.MetaData)))
	for xvk, bzg := range z.MetaData {
		o = msgp.AppendString(o, xvk)
		o, err = msgp.AppendIntf(o, bzg)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ItemMeta) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "_id":
			{
				var tmp uint64
				tmp, bts, err = msgp.ReadUint64Bytes(bts)
				z.ItemId = ItemId(tmp)
			}
			if err != nil {
				return
			}
		case "_filepath":
			z.FilePath, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "metadata":
			var msz uint32
			msz, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.MetaData == nil && msz > 0 {
				z.MetaData = make(map[string]interface{}, msz)
			} else if len(z.MetaData) > 0 {
				for key, _ := range z.MetaData {
					delete(z.MetaData, key)
				}
			}
			for msz > 0 {
				var xvk string
				var bzg interface{}
				msz--
				xvk, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				bzg, bts, err = msgp.ReadIntfBytes(bts)
				if err != nil {
					return
				}
				z.MetaData[xvk] = bzg
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *ItemMeta) Msgsize() (s int) {
	s = msgp.MapHeaderSize + msgp.StringPrefixSize + 3 + msgp.Uint64Size + msgp.StringPrefixSize + 9 + msgp.StringPrefixSize + len(z.FilePath) + msgp.StringPrefixSize + 8 + msgp.MapHeaderSize
	if z.MetaData != nil {
		for xvk, bzg := range z.MetaData {
			_ = bzg
			s += msgp.StringPrefixSize + len(xvk) + msgp.GuessSize(bzg)
		}
	}
	return
}
