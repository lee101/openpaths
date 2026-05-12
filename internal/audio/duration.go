package audio

import (
	"bytes"
	"encoding/binary"
)

const fallbackDurationSeconds = 60

// EstimateDurationSeconds returns the audio duration when it can be cheaply
// inferred from the container. Unknown formats fall back to one billable minute.
func EstimateDurationSeconds(filename string, data []byte) int {
	if seconds := wavDurationSeconds(data); seconds > 0 {
		return seconds
	}
	return fallbackDurationSeconds
}

func wavDurationSeconds(data []byte) int {
	if len(data) < 12 || !bytes.Equal(data[:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return 0
	}

	var byteRate uint32
	var dataSize uint32
	for pos := 12; pos+8 <= len(data); {
		chunkID := data[pos : pos+4]
		chunkSize := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		chunkStart := pos + 8
		chunkEnd := chunkStart + int(chunkSize)
		if chunkEnd > len(data) {
			return 0
		}

		switch {
		case bytes.Equal(chunkID, []byte("fmt ")) && chunkSize >= 16:
			byteRate = binary.LittleEndian.Uint32(data[chunkStart+8 : chunkStart+12])
		case bytes.Equal(chunkID, []byte("data")):
			dataSize = chunkSize
		}

		pos = chunkEnd
		if chunkSize%2 == 1 {
			pos++
		}
	}

	if byteRate == 0 || dataSize == 0 {
		return 0
	}
	return int((dataSize + byteRate - 1) / byteRate)
}
