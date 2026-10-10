package email

import (
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func parseBuilt(t *testing.T, raw []byte) (*mail.Message, map[string]string) {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	mt, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/alternative" {
		t.Fatalf("content-type %q %v", mt, err)
	}
	parts := map[string]string{}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(p)
		if err != nil {
			t.Fatal(err)
		}
		ct, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		parts[ct] = string(b)
	}
	if !strings.Contains(string(raw), "Content-Transfer-Encoding: quoted-printable") {
		t.Fatal("parts not quoted-printable")
	}
	return msg, parts
}

func TestBuildMessageHeadersAndParts(t *testing.T) {
	html := `<html><head><style>p{}</style></head><body><p>Hello <b>there</b> &amp; welcome 日本</p><p><a href="https://openpaths.io/x">Docs</a></p></body></html>` + strings.Repeat("x", 3000)
	rcpt, raw, err := BuildMessage("from@openpaths.io", Message{
		To:             "User@Example.com",
		Subject:        "Hi\r\nBcc: evil@x.com — live",
		HTML:           html,
		UnsubscribeURL: "https://openpaths.io/unsubscribe?e=a&t=b",
		FromName:       "Open Paths é",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rcpt != "User@Example.com" {
		t.Fatalf("rcpt %q", rcpt)
	}
	msg, parts := parseBuilt(t, raw)
	if msg.Header.Get("Bcc") != "" {
		t.Fatal("header injection via subject")
	}
	subj, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil || subj != "Hi Bcc: evil@x.com — live" {
		t.Fatalf("subject %q %v", subj, err)
	}
	if strings.ContainsAny(msg.Header.Get("Subject"), "—") {
		t.Fatal("subject not RFC2047 encoded")
	}
	from, err := mail.ParseAddress(msg.Header.Get("From"))
	if err != nil || from.Name != "Open Paths é" || from.Address != "from@openpaths.io" {
		t.Fatalf("from %+v %v", from, err)
	}
	if id := msg.Header.Get("Message-Id"); !strings.HasPrefix(id, "<") || !strings.HasSuffix(id, "@openpaths.io>") {
		t.Fatalf("message-id %q", id)
	}
	if msg.Header.Get("Precedence") != "bulk" || msg.Header.Get("List-Unsubscribe-Post") != "List-Unsubscribe=One-Click" {
		t.Fatal("bulk headers missing")
	}
	if msg.Header.Get("List-Unsubscribe") != "<https://openpaths.io/unsubscribe?e=a&t=b>" {
		t.Fatalf("list-unsubscribe %q", msg.Header.Get("List-Unsubscribe"))
	}
	if msg.Header.Get("Date") == "" {
		t.Fatal("no date")
	}
	text := parts["text/plain"]
	if !strings.Contains(text, "Hello there & welcome 日本") || !strings.Contains(text, "Docs (https://openpaths.io/x)") || strings.Contains(text, "<") || strings.Contains(text, "p{}") {
		t.Fatalf("plain part: %q", text)
	}
	if !strings.Contains(text, "Unsubscribe: https://openpaths.io/unsubscribe") {
		t.Fatalf("plain part lacks unsubscribe: %q", text)
	}
	if parts["text/html"] != html {
		t.Fatal("html part not round-tripped")
	}
	if strings.Contains(string(raw), "\n\n") && !strings.Contains(string(raw), "\r\n\r\n") {
		t.Fatal("bare LF")
	}
	for _, l := range strings.Split(string(raw), "\r\n") {
		if len(l) > 998 {
			t.Fatalf("line too long: %d", len(l))
		}
	}
}

func TestBuildMessageTransactionalNotBulk(t *testing.T) {
	_, raw, err := BuildMessage("from@openpaths.io", Message{To: "a@b.com", Subject: "Low balance", HTML: "<p>hi</p>"})
	if err != nil {
		t.Fatal(err)
	}
	msg, _ := parseBuilt(t, raw)
	if msg.Header.Get("Precedence") != "" || msg.Header.Get("List-Unsubscribe") != "" {
		t.Fatal("transactional mail marked bulk")
	}
	if got := msg.Header.Get("Subject"); got != "Low balance" {
		t.Fatalf("ascii subject altered: %q", got)
	}
	if msg.Header.Get("From") != "\"OpenPaths\" <from@openpaths.io>" && msg.Header.Get("From") != "OpenPaths <from@openpaths.io>" {
		t.Fatalf("from %q", msg.Header.Get("From"))
	}
}

func TestBuildMessageRejectsBadRecipients(t *testing.T) {
	for _, to := range []string{"a@b.com\r\nBcc: x@y.com", "not an address", ""} {
		if _, _, err := BuildMessage("f@o.io", Message{To: to, Subject: "s", HTML: "x"}); err == nil {
			t.Errorf("accepted %q", to)
		}
	}
	if _, _, err := BuildMessage("f@o.io", Message{To: "a@b.com", Subject: "s", HTML: "x", UnsubscribeURL: "http://x>\r\nBcc: y"}); err == nil {
		t.Error("accepted bad unsubscribe url")
	}
}
