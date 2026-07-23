package mailer

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
)

// Config from env:
//
//	HUB_SMTP_HOST, HUB_SMTP_PORT (default 587), HUB_SMTP_USER, HUB_SMTP_PASS, HUB_SMTP_FROM
//	HUB_PUBLIC_URL (default https://hub.stifer.xyz)
//	HUB_MAIL_MODE = smtp | log  (log = print to journal, no real send; default smtp if host set else log)
//
// QQ 邮箱示例：
//
//	HUB_SMTP_HOST=smtp.qq.com
//	HUB_SMTP_PORT=465   # 或 587
//	HUB_SMTP_USER=你的QQ号@qq.com
//	HUB_SMTP_PASS=授权码（不是QQ密码）
//	HUB_SMTP_FROM=你的QQ号@qq.com
type Config struct {
	Host      string
	Port      string
	User      string
	Pass      string
	From      string
	PublicURL string
	Mode      string // smtp | log
}

func Load() Config {
	host := strings.TrimSpace(os.Getenv("HUB_SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("HUB_SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	mode := strings.TrimSpace(os.Getenv("HUB_MAIL_MODE"))
	if mode == "" {
		if host != "" {
			mode = "smtp"
		} else {
			mode = "log"
		}
	}
	from := strings.TrimSpace(os.Getenv("HUB_SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("HUB_SMTP_USER"))
	}
	pub := strings.TrimSpace(os.Getenv("HUB_PUBLIC_URL"))
	if pub == "" {
		pub = "https://hub.stifer.xyz"
	}
	return Config{
		Host:      host,
		Port:      port,
		User:      strings.TrimSpace(os.Getenv("HUB_SMTP_USER")),
		Pass:      os.Getenv("HUB_SMTP_PASS"),
		From:      from,
		PublicURL: strings.TrimRight(pub, "/"),
		Mode:      mode,
	}
}

func (c Config) CanSend() bool {
	if c.Mode == "log" {
		return true
	}
	return c.Host != "" && c.From != ""
}

func (c Config) buildMessage(to, subject, textBody, htmlBody string) []byte {
	boundary := "hub-mail-boundary"
	var msg strings.Builder
	msg.WriteString("From: " + c.From + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	if htmlBody != "" {
		msg.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n\r\n")
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(textBody + "\r\n")
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		msg.WriteString(htmlBody + "\r\n")
		msg.WriteString("--" + boundary + "--\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(textBody)
	}
	return []byte(msg.String())
}

// Send plain text (+ simple HTML) email.
// Port 465: implicit TLS (QQ 常用). Port 587 / others: STARTTLS via net/smtp.SendMail.
func (c Config) Send(to, subject, textBody, htmlBody string) error {
	if c.Mode == "log" || c.Host == "" {
		log.Printf("[mail:log] to=%s subject=%q body=%s", to, subject, textBody)
		return nil
	}
	if c.From == "" {
		return fmt.Errorf("HUB_SMTP_FROM not set")
	}

	body := c.buildMessage(to, subject, textBody, htmlBody)
	addr := c.Host + ":" + c.Port

	var auth smtp.Auth
	if c.User != "" {
		auth = smtp.PlainAuth("", c.User, c.Pass, c.Host)
	}

	// Implicit SSL (QQ / 网易 / 阿里云 常见 465)
	var sendErr error
	if c.Port == "465" {
		sendErr = sendSMTPS(addr, c.Host, c.From, []string{to}, body, auth)
	} else {
		sendErr = smtp.SendMail(addr, auth, c.From, []string{to}, body)
	}
	if sendErr != nil {
		log.Printf("[mail:smtp] FAIL to=%s host=%s:%s err=%v", to, c.Host, c.Port, sendErr)
		return sendErr
	}
	log.Printf("[mail:smtp] OK to=%s subject=%q via %s:%s", to, subject, c.Host, c.Port)
	return nil
}

func sendSMTPS(addr, host, from string, to []string, msg []byte, auth smtp.Auth) error {
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// Ensure net is referenced if needed by older builds (keepalive unused).
var _ = net.Dial

func NewToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func VerifyEmailContent(publicURL, rawToken string) (subject, text, html string) {
	link := publicURL + "/verify-email?token=" + rawToken
	subject = "Verify your Agent Hub email"
	text = "Welcome to Agent Hub.\n\nClick to verify your email (valid 24h):\n" + link + "\n\nIf you did not register, ignore this email.\n"
	html = `<p>Welcome to <b>Agent Hub</b>.</p>
<p><a href="` + link + `">Click here to verify your email</a> (valid 24 hours).</p>
<p>If you did not register, ignore this email.</p>`
	return
}
