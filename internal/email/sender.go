package email

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html"
	"log"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"time"
)

var Suppressed func(email string) bool

type Message struct {
	To             string
	Subject        string
	HTML           string
	Text           string
	UnsubscribeURL string
	FromName       string
}

func Send(toEmail, subject, htmlBody string) error {
	return SendWithUnsubscribe(toEmail, subject, htmlBody, "")
}

func SendWithUnsubscribe(toEmail, subject, htmlBody, unsubscribeURL string) error {
	return SendMessage(Message{To: toEmail, Subject: subject, HTML: htmlBody, UnsubscribeURL: unsubscribeURL})
}

func SendMessage(m Message) error {
	if Suppressed != nil && Suppressed(m.To) {
		log.Printf("email suppressed, not sending to %s: %s", m.To, m.Subject)
		return nil
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	smtpUser := os.Getenv("AWS_SMTP_USERNAME")
	smtpPass := os.Getenv("AWS_SMTP_PASSWORD")
	if smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("AWS_SMTP_USERNAME and AWS_SMTP_PASSWORD required")
	}

	smtpHost := fmt.Sprintf("email-smtp.%s.amazonaws.com", region)

	fromEmail := os.Getenv("SES_FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = "lee.penkman@netwrck.com"
	}
	if m.FromName == "" {
		m.FromName = os.Getenv("SES_FROM_NAME")
	}

	rcpt, msg, err := BuildMessage(fromEmail, m)
	if err != nil {
		return err
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	tlsConfig := &tls.Config{ServerName: smtpHost}
	conn, err := tls.Dial("tcp", smtpHost+":465", tlsConfig)
	if err != nil {
		log.Printf("TLS direct failed, trying STARTTLS on :587: %v", err)
		err = smtp.SendMail(smtpHost+":587", auth, fromEmail, []string{rcpt}, msg)
		if err != nil {
			return fmt.Errorf("SMTP send failed: %w", err)
		}
		log.Printf("email sent to %s via STARTTLS: %s", rcpt, m.Subject)
		return nil
	}

	c, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP client error: %w", err)
	}
	defer c.Close()

	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}
	if err = c.Mail(fromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err = c.Rcpt(rcpt); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}
	c.Quit()

	log.Printf("email sent to %s via TLS: %s", rcpt, m.Subject)
	return nil
}

func SanitizeHeader(s string) string {
	s = reLineBreaks.ReplaceAllString(strings.ReplaceAll(s, "\x00", ""), " ")
	return strings.TrimSpace(s)
}

func encodeSubject(s string) string {
	return mime.QEncoding.Encode("utf-8", SanitizeHeader(s))
}

func BuildMessage(fromEmail string, m Message) (string, []byte, error) {
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.UnsubscribeURL, "\r\n<>") {
		return "", nil, fmt.Errorf("invalid header characters in recipient or unsubscribe url")
	}
	to, err := mail.ParseAddress(strings.TrimSpace(m.To))
	if err != nil {
		return "", nil, fmt.Errorf("invalid recipient %q: %w", m.To, err)
	}
	fromName := SanitizeHeader(m.FromName)
	if fromName == "" {
		fromName = "OpenPaths"
	}
	from := (&mail.Address{Name: fromName, Address: fromEmail}).String()

	domain := "openpaths.io"
	if i := strings.LastIndex(fromEmail, "@"); i >= 0 && i < len(fromEmail)-1 {
		domain = fromEmail[i+1:]
	}

	text := m.Text
	if text == "" {
		text = HTMLToText(m.HTML)
	}
	if m.UnsubscribeURL != "" && !strings.Contains(text, m.UnsubscribeURL) {
		text += "\n\nUnsubscribe: " + m.UnsubscribeURL + "\n"
	}

	boundary := "----=_Part_" + randomHex(8)
	var b bytes.Buffer
	hdr := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	hdr("From", from)
	hdr("To", to.Address)
	hdr("Subject", encodeSubject(m.Subject))
	hdr("Date", time.Now().Format(time.RFC1123Z))
	hdr("Message-ID", "<"+randomHex(16)+"@"+domain+">")
	if m.UnsubscribeURL != "" {
		hdr("List-Unsubscribe", "<"+m.UnsubscribeURL+">")
		hdr("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
		hdr("Precedence", "bulk")
	}
	hdr("MIME-Version", "1.0")
	hdr("Content-Type", "multipart/alternative; boundary=\""+boundary+"\"")
	b.WriteString("\r\n")
	for _, p := range []struct{ ctype, body string }{{"text/plain", text}, {"text/html", m.HTML}} {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: " + p.ctype + "; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n")
		w := quotedprintable.NewWriter(&b)
		w.Write([]byte(p.body))
		w.Close()
		b.WriteString("\r\n")
	}
	b.WriteString("--" + boundary + "--\r\n")
	return to.Address, b.Bytes(), nil
}

var (
	reLineBreaks = regexp.MustCompile(`[\r\n]+`)
	reHead       = regexp.MustCompile(`(?is)<(head|style|script)\b.*?</(head|style|script)>`)
	reAnchor     = regexp.MustCompile(`(?is)<a\b[^>]*?href\s*=\s*"([^"]*)"[^>]*>(.*?)</a>`)
	reBreak      = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|tr|h[1-6]|li|table)>`)
	reCell       = regexp.MustCompile(`(?i)</t[dh]>`)
	reTag        = regexp.MustCompile(`(?s)<[^>]*>`)
	reSpaces     = regexp.MustCompile(`[ \t\f\v]+`)
	reLines      = regexp.MustCompile(`\n{3,}`)
)

func HTMLToText(src string) string {
	s := reHead.ReplaceAllString(src, "")
	s = reAnchor.ReplaceAllStringFunc(s, func(a string) string {
		sm := reAnchor.FindStringSubmatch(a)
		label := strings.TrimSpace(reTag.ReplaceAllString(sm[2], ""))
		href := html.UnescapeString(sm[1])
		if !strings.HasPrefix(href, "http") || label == "" || label == href {
			return label
		}
		return label + " (" + href + ")"
	})
	s = reBreak.ReplaceAllString(s, "\n")
	s = reCell.ReplaceAllString(s, " ")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, " ", " ")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(reSpaces.ReplaceAllString(l, " "))
	}
	s = reLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimSpace(s) + "\n"
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
