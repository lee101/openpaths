package gobed

import (
	"encoding/base64"
	"encoding/binary"
	"math"
)

func encodeF32(v []float32) string {
	raw := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(raw[4*i:], math.Float32bits(x))
	}
	return base64.StdEncoding.EncodeToString(raw)
}
