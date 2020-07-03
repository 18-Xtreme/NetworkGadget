package utils

import (
	"encoding/binary"
)

func Int16ToBytes(i int16) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf, uint16(i))
	return buf
}

func BytesToInt16(buf []byte) int16 {
	return int16(binary.BigEndian.Uint16(buf))
}
