package provider

import (
	"net/http"
	"testing"
)

func TestTuneDefaultTransport(t *testing.T) {
	TuneDefaultTransport()
	tr := http.DefaultTransport.(*http.Transport)
	if tr.MaxIdleConnsPerHost < 64 || tr.MaxIdleConns < tr.MaxIdleConnsPerHost {
		t.Fatalf("idle limits not raised: %d/%d", tr.MaxIdleConns, tr.MaxIdleConnsPerHost)
	}
}
