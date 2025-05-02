/*
 * TinyMail - Minimalistic email client
 * Copyright (C) 2025 Cyanic
 *
 * This program is free software: you can redistribute it and/or modify it
 * under the terms of the GNU Affero General Public License as published by the
 * Free Software Foundation, either version 3 of the License, or (at your
 * option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
 * FITNESS FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License
 * for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package smtp

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type client struct {
	Email    string
	Password string
	Domain   string
	Host     string
	Port     int
}

func NewClient(email, password string) (c client, err error) {
	domain, err := emailDomain(email)
	if err != nil {
		return
	}
	c = client{
		Email:    email,
		Password: password,
		Domain:   domain,
	}
	return
}

func emailDomain(email string) (string, error) {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return "", fmt.Errorf("invalid email address format")
	}
	return email[at+1:], nil
}

func (c client) WithServer(host string, port int) client {
	return client{
		c.Email,
		c.Password,
		c.Domain,
		host,
		port,
	}
}

func sanitize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
}

func toRFC2047(s string) string {
	return mime.QEncoding.Encode("utf-8", s)
}

func newMessageID(domain string) (mid string, err error) {
	unix := time.Now().UTC().Unix()
	rnd := make([]byte, 16)
	_, err = rand.Read(rnd)
	if err != nil {
		return
	}
	return fmt.Sprintf("<%d.%x@%s>", unix, rnd, domain), nil
}

func encodeBody(body string, buf *bytes.Buffer) (err error) {
	encoder := quotedprintable.NewWriter(buf)
	_, err = encoder.Write([]byte(body))
	if err != nil {
		return
	}
	encoder.Close()
	return
}

func (c client) send(from, to, subject, body string) (err error) {
	from = sanitize(from)
	to = sanitize(to)
	subject = sanitize(subject)

	mid, err := newMessageID(c.Domain)
	if err != nil {
		return
	}

	fromAddr, err := mail.ParseAddress(from)
	if err != nil {
		return
	}
	toAddr, err := mail.ParseAddress(to)
	if err != nil {
		return
	}

	date := time.Now().UTC().Format(time.RFC1123Z)
	headers := map[string]string{
		"From":                      fromAddr.String(),
		"To":                        toAddr.String(),
		"Subject":                   toRFC2047(subject),
		"Date":                      date,
		"Message-ID":                mid,
		"MIME-Version":              "1.0",
		"Content-Type":              "text/plain; charset=utf-8",
		"Content-Transfer-Encoding": "quoted-printable",
	}

	var buf bytes.Buffer
	for k, v := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	buf.WriteString("\r\n")

	err = encodeBody(body, &buf)
	if err != nil {
		return
	}

	err = smtp.SendMailTLS(
		fmt.Sprintf("%s:%d", c.Host, c.Port),
		sasl.NewPlainClient("", c.Email, c.Password),
		fromAddr.Address,
		[]string{toAddr.Address},
		bytes.NewReader(buf.Bytes()),
	)
	if err != nil {
		return
	}

	return
}

func (c client) Send(from, to, subject, body string) (err error) {
	if c.Host != "" {
		return c.send(from, to, subject, body)
	}

	_, srvs, err := net.LookupSRV("submissions", "tcp", c.Domain)
	if err != nil {
		return
	}

	for _, srv := range srvs {
		target := strings.TrimSuffix(srv.Target, ".")
		if target == "" {
			err = errors.New("SRV record does not exist")
			continue
		}

		client := c.WithServer(target, int(srv.Port))
		err = client.Send(from, to, subject, body)
		if err == nil {
			return
		}
	}
	return
}
