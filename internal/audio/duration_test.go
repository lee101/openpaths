package audio

import (
	"encoding/binary"
	"testing"
)

func TestEstimateDurationSeconds_WAV(t *testing.T) {
	data := makeTestWAV(16000, 32000)
	got := EstimateDurationSeconds("sample.wav", data)
	if got != 2 {
		t.Fatalf("duration = %d, want 2", got)
	}
}

func TestEstimateDurationSeconds_Fallback(t *testing.T) {
	got := EstimateDurationSeconds("sample.mp3", []byte("not a parseable mp3"))
	if got != fallbackDurationSeconds {
		t.Fatalf("duration = %d, want %d", got, fallbackDurationSeconds)
	}
}

func makeTestWAV(byteRate, dataSize uint32) []byte {
	data := make([]byte, 44+dataSize)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	copy(data[8:12], "WAVE")
	copy(data[12:16], "fmt ")
	binary.LittleEndian.PutUint32(data[16:20], 16)
	binary.LittleEndian.PutUint16(data[20:22], 1)
	binary.LittleEndian.PutUint16(data[22:24], 1)
	binary.LittleEndian.PutUint32(data[24:28], 16000)
	binary.LittleEndian.PutUint32(data[28:32], byteRate)
	binary.LittleEndian.PutUint16(data[32:34], 2)
	binary.LittleEndian.PutUint16(data[34:36], 16)
	copy(data[36:40], "data")
	binary.LittleEndian.PutUint32(data[40:44], dataSize)
	return data
}
