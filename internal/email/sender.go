package email

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
	"os"
)

// Suppressed is an optional hook consulted before every send; wire it to the
// email_suppressions table so unsubscribed/blocked addresses never get mail.
var Suppressed func(email string) bool

// Send sends an HTML email via AWS SES SMTP.
func Send(toEmail, subject, htmlBody string) error {
	return SendWithUnsubscribe(toEmail, subject, htmlBody, "")
}

// SendWithUnsubscribe sends an HTML email with RFC 8058 List-Unsubscribe headers.
func SendWithUnsubscribe(toEmail, subject, htmlBody, unsubscribeURL string) error {
	if Suppressed != nil && Suppressed(toEmail) {
		log.Printf("email suppressed, not sending to %s: %s", toEmail, subject)
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

	boundary := "----=_Part_" + randomHex(8)
	plainText := "View this email in your browser."

	unsubHeaders := ""
	if unsubscribeURL != "" {
		unsubHeaders = fmt.Sprintf("List-Unsubscribe: <%s>\r\nList-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n", unsubscribeURL)
	}

	msg := fmt.Sprintf(
		"From: OpenPaths <%s>\r\nTo: %s\r\nSubject: %s\r\n%sMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n--%s--\r\n",
		fromEmail, toEmail, subject, unsubHeaders, boundary, boundary, plainText, boundary, htmlBody, boundary,
	)

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Try TLS direct on port 465 first, fall back to STARTTLS on 587.
	tlsConfig := &tls.Config{ServerName: smtpHost}
	conn, err := tls.Dial("tcp", smtpHost+":465", tlsConfig)
	if err != nil {
		log.Printf("TLS direct failed, trying STARTTLS on :587: %v", err)
		err = smtp.SendMail(smtpHost+":587", auth, fromEmail, []string{toEmail}, []byte(msg))
		if err != nil {
			return fmt.Errorf("SMTP send failed: %w", err)
		}
		log.Printf("email sent to %s via STARTTLS: %s", toEmail, subject)
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
	if err = c.Rcpt(toEmail); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}
	c.Quit()

	log.Printf("email sent to %s via TLS: %s", toEmail, subject)
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
