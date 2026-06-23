package savedresp

import (
	"math"
	"testing"
)

func TestEncodeDecodeRoundTripNormalizes(t *testing.T) {
	in := []float64{3, 4, 0} // L2 norm = 5
	buf := encodeVec(in)
	if len(buf) != len(in)*4 {
		t.Fatalf("expected %d bytes, got %d", len(in)*4, len(buf))
	}
	out := decodeVec(buf)
	if len(out) != len(in) {
		t.Fatalf("expected %d floats, got %d", len(in), len(out))
	}
	// Normalized: 3/5, 4/5, 0
	want := []float32{0.6, 0.8, 0}
	for i := range want {
		if math.Abs(float64(out[i]-want[i])) > 1e-6 {
			t.Errorf("out[%d] = %v, want %v", i, out[i], want[i])
		}
	}
}

func TestDecodeRejectsBadLength(t *testing.T) {
	if got := decodeVec([]byte{1, 2, 3}); got != nil {
		t.Errorf("expected nil for non-multiple-of-4 input, got %v", got)
	}
	if got := decodeVec(nil); got != nil {
		t.Errorf("expected nil for empty input, got %v", got)
	}
}

func TestDotCosineOfNormalizedVectors(t *testing.T) {
	a := normalize([]float64{1, 0, 0})
	b := normalize([]float64{1, 0, 0})
	if d := dot(a, b); math.Abs(d-1) > 1e-6 {
		t.Errorf("identical unit vectors should have cosine 1, got %v", d)
	}
	c := normalize([]float64{0, 1, 0})
	if d := dot(a, c); math.Abs(d) > 1e-6 {
		t.Errorf("orthogonal unit vectors should have cosine 0, got %v", d)
	}
	// Round-trip through bytea must preserve the score.
	bb := decodeVec(encodeVec([]float64{1, 0, 0}))
	if d := dot(a, bb); math.Abs(d-1) > 1e-6 {
		t.Errorf("decoded vector cosine should be 1, got %v", d)
	}
}
