package gobed

import (
	"encoding/json"
	"fmt"
	"os"
)

const maxWordBytes = 100

// fastTokenizer is an allocation-free ASCII WordPiece tokenizer producing the
// same ids as gobed's SimpleTokenize for ASCII text (words over maxWordBytes
// map to [UNK], as in BERT).
type fastTokenizer struct {
	vocab         map[string]int16
	cls, sep, unk int16
}

func loadFastTokenizer(path string) (*fastTokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var td struct {
		Model struct {
			Vocab map[string]int `json:"vocab"`
		} `json:"model"`
	}
	if err := json.Unmarshal(data, &td); err != nil {
		return nil, err
	}
	vocab := make(map[string]int16, len(td.Model.Vocab))
	for tok, id := range td.Model.Vocab {
		if id < 32768 {
			vocab[tok] = int16(id)
		}
	}
	t := &fastTokenizer{vocab: vocab, cls: -1, sep: -1, unk: -1}
	if id, ok := vocab["[CLS]"]; ok {
		t.cls = id
	}
	if id, ok := vocab["[SEP]"]; ok {
		t.sep = id
	}
	if id, ok := vocab["[UNK]"]; ok {
		t.unk = id
	}
	if t.unk < 0 {
		return nil, fmt.Errorf("vocab has no [UNK]")
	}
	return t, nil
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func asciiWord(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '\'' || c == '_'
}

func asciiSpace(c byte) bool {
	return c == ' ' || c >= '\t' && c <= '\r'
}

// tokenize appends ids for ASCII text to dst, stopping once len(dst) reaches
// limit (the trailing [SEP] is dropped in that case, matching truncation).
func (t *fastTokenizer) tokenize(text string, dst []int16, limit int) []int16 {
	if t.cls >= 0 {
		dst = append(dst, t.cls)
	}
	var buf [maxWordBytes + 2]byte
	var pieces [maxWordBytes]int16
	n := len(text)
	for i := 0; i < n && len(dst) < limit; {
		c := text[i]
		if asciiSpace(c) {
			i++
			continue
		}
		if !asciiWord(c) {
			if id, ok := t.vocab[text[i:i+1]]; ok {
				dst = append(dst, id)
			} else {
				dst = append(dst, t.unk)
			}
			i++
			continue
		}
		j := i + 1
		for j < n && asciiWord(text[j]) {
			j++
		}
		wl := j - i
		if wl > maxWordBytes {
			dst = append(dst, t.unk)
			i = j
			continue
		}
		w := buf[2 : 2+wl]
		for k := 0; k < wl; k++ {
			b := text[i+k]
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			w[k] = b
		}
		i = j
		if id, ok := t.vocab[string(w)]; ok {
			dst = append(dst, id)
			continue
		}
		np := 0
		start := 0
		ok := true
		for start < wl {
			p := 2 + start
			lo := p
			if start > 0 {
				buf[p-2], buf[p-1] = '#', '#'
				lo = p - 2
			}
			end := wl
			matched := false
			for ; end > start; end-- {
				if id, hit := t.vocab[string(buf[lo:2+end])]; hit {
					pieces[np] = id
					np++
					matched = true
					break
				}
			}
			if !matched {
				ok = false
				break
			}
			start = end
		}
		if ok {
			dst = append(dst, pieces[:np]...)
		} else {
			dst = append(dst, t.unk)
		}
	}
	if len(dst) < limit && t.sep >= 0 {
		dst = append(dst, t.sep)
	}
	if len(dst) > limit {
		dst = dst[:limit]
	}
	return dst
}
