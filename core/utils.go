package core

import (
	"bytes"
	"encoding/binary"
)

func IntToHex(n int64) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, n)
	return buf.Bytes()
}