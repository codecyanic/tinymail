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

package imap

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"slices"
	"sort"
	"strings"

	"github.com/codecyanic/tinymail/internal/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/mnako/letters"
)

const messageLimit = 500
const pageSize = 25

type client struct {
	Email    string
	Password string
}

func NewClient(email, password string) client {
	return client{email, password}
}

func emailDomain(email string) (string, error) {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return "", fmt.Errorf("invalid email address format")
	}
	return email[at+1:], nil
}

type imapSession struct {
	*imapclient.Client
}

func (s imapSession) Close() error {
	defer s.Client.Close()
	if err := s.Logout().Wait(); err != nil {
		return err
	}
	return nil
}

func (c client) Session() (session imapSession, err error) {
	domain, err := emailDomain(c.Email)
	if err != nil {
		return
	}

	_, srvs, err := net.LookupSRV("imaps", "tcp", domain)
	if err != nil {
		return
	}

	for _, srv := range srvs {
		target := strings.TrimSuffix(srv.Target, ".")
		addr := fmt.Sprintf("%s:%d", target, srv.Port)
		session.Client, err = imapclient.DialTLS(addr, nil)
		if err == nil {
			break
		}
		log.Printf("failed to dial IMAP server: %v", err)
	}
	if err != nil {
		return
	}

	err = session.Client.Login(c.Email, c.Password).Wait()
	if err != nil {
		session.Client.Close()
	}
	return
}

func (c client) Account() (acc models.Account, err error) {
	session, err := c.Session()
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, session.Close())
	}()

	mailboxes, err := session.List("", "%", nil).Collect()
	if err != nil {
		return
	}

	// Ensure INBOX is the first mailbox
	acc.Mailboxes = []models.Mailbox{
		models.Mailbox{Name: "INBOX"},
	}
	for _, mbox := range mailboxes {
		if mbox.Mailbox != "INBOX" {
			acc.Mailboxes = append(acc.Mailboxes, models.Mailbox{
				Name: mbox.Mailbox,
			})
		}
	}
	return
}

func (s imapSession) fetchUIDs(start, end uint32) (uids []imap.UID, err error) {
	if start > end {
		return nil, fmt.Errorf(
			"start must not be larger than end: %v > %v", start, end,
		)
	}
	count := end - start
	nums := make([]uint32, count)
	for i := range count {
		nums[i] = start + i + 1
	}
	seqSet := imap.SeqSetNum(nums...)
	fetchOptions := &imap.FetchOptions{UID: true}
	messages, err := s.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		return
	}

	uids = make([]imap.UID, len(messages))
	for i, msg := range messages {
		uids[i] = msg.UID
	}
	return
}

func (s imapSession) fetchMessages(uids []imap.UID) (messages []models.Message, err error) {
	messages = make([]models.Message, 0, len(uids))

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
		UID:      true,
	}
	uidSet := imap.UIDSetNum(uids...)
	fetchedMessages, err := s.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return
	}
	for _, msg := range fetchedMessages {
		var from string
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Addr()
		}
		messages = append(messages, models.Message{
			UID:     uint32(msg.UID),
			Seen:    slices.Contains(msg.Flags, imap.FlagSeen),
			From:    from,
			Subject: msg.Envelope.Subject,
		})
	}
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].UID > messages[j].UID
	})
	return
}

func (c client) Mailbox(name string) (mbx models.Mailbox, err error) {
	mbx.Name = name

	session, err := c.Session()
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, session.Close())
	}()

	selected, err := session.Select(name, nil).Wait()
	if err != nil {
		return
	}
	if selected.NumMessages == 0 {
		mbx.UIDs = []uint32{}
		mbx.Messages = []models.Message{}
		return
	}

	uidSet := make(map[imap.UID]any)
	for start := uint32(0); start < selected.NumMessages; start += messageLimit {
		end := start + messageLimit
		if end > selected.NumMessages {
			end = selected.NumMessages
		}
		batch, err := session.fetchUIDs(start, end)
		if err != nil {
			return mbx, err
		}
		for _, uid := range batch {
			uidSet[uid] = struct{}{}
		}
	}

	uids := make([]imap.UID, 0, len(uidSet))
	for key := range uidSet {
		uids = append(uids, key)
	}
	sort.Slice(uids, func(i, j int) bool {
		return uids[i] > uids[j]
	})
	mbx.UIDs = make([]uint32, len(uids))
	for i, uid := range uids {
		mbx.UIDs[i] = uint32(uid)
	}

	var page []imap.UID
	if len(uids) > pageSize {
		page = uids[:pageSize]
	} else {
		page = uids
	}

	mbx.Messages, err = session.fetchMessages(page)
	if err != nil {
		return
	}
	return
}

func (c client) Messages(mailbox string, uids []uint32) (messages []models.Message, err error) {
	session, err := c.Session()
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, session.Close())
	}()

	_, err = session.Select(mailbox, nil).Wait()
	if err != nil {
		err = fmt.Errorf("failed to select %s: %v", mailbox, err)
		return
	}

	UIDs := make([]imap.UID, len(uids))
	for i, uid := range uids {
		UIDs[i] = imap.UID(uid)
	}
	return session.fetchMessages(UIDs)
}

func (s imapSession) fetchBodyStructure(uid imap.UID) (bs imap.BodyStructure, err error) {
	fetchOptions := &imap.FetchOptions{
		BodyStructure: &imap.FetchItemBodyStructure{},
	}
	uidSet := imap.UIDSetNum(uid)
	messages, err := s.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return
	}
	if len(messages) == 0 {
		err = fmt.Errorf("message not found")
		return
	}
	return messages[0].BodyStructure, nil
}

func findPlainTextPath(bs imap.BodyStructure) (plainTextPath []int) {
	bs.Walk(func(path []int, part imap.BodyStructure) (walkChildren bool) {
		// Walk would get called for siblings after the part was found
		if plainTextPath != nil {
			return false
		}

		if part.MediaType() == "text/plain" {
			plainTextPath = path
			return false
		}

		forwardedChildren := part.MediaType() == "message/rfc822"
		return !forwardedChildren
	})
	return
}

func decodeMessagePart(mime, text []byte) (string, error) {
	reader := bytes.NewReader(append(mime, text...))

	email, err := letters.ParseEmail(reader)
	if err != nil {
		return "", err
	}
	return email.Text, nil
}

func (s imapSession) fetchMessagePart(uid imap.UID, path []int) (
	message models.Message, err error,
) {
	mimeSection := &imap.FetchItemBodySection{
		Part:      path,
		Specifier: imap.PartSpecifierMIME,
	}
	textSection := &imap.FetchItemBodySection{
		Part: path,
	}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{mimeSection, textSection},
		Envelope:    true,
		Flags:       true,
		UID:         true,
	}
	uidSet := imap.UIDSetNum(imap.UID(uid))
	messages, err := s.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return
	}
	if len(messages) == 0 {
		err = fmt.Errorf("message not found")
		return
	}
	msg := messages[0]
	body, err := decodeMessagePart(
		msg.FindBodySection(mimeSection),
		msg.FindBodySection(textSection),
	)
	if err != nil {
		return
	}

	var from string
	if len(msg.Envelope.From) > 0 {
		from = msg.Envelope.From[0].Addr()
	}

	message = models.Message{
		UID:     uint32(msg.UID),
		Seen:    slices.Contains(msg.Flags, imap.FlagSeen),
		From:    from,
		Subject: msg.Envelope.Subject,
		Body:    body,
	}
	return
}

func (c client) Message(mailbox string, uid uint32) (message models.Message, err error) {
	UID := imap.UID(uid)

	session, err := c.Session()
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, session.Close())
	}()

	_, err = session.Select(mailbox, nil).Wait()
	if err != nil {
		err = fmt.Errorf("failed to select %s: %v", mailbox, err)
		return
	}

	bs, err := session.fetchBodyStructure(UID)
	if err != nil {
		return
	}
	path := findPlainTextPath(bs)
	if path == nil {
		uids := []imap.UID{UID}
		var messages []models.Message
		messages, err = session.fetchMessages(uids)
		if err != nil {
			return
		}
		if len(messages) == 0 {
			err = fmt.Errorf("message not found")
			return
		}
		return messages[0], nil
	}

	return session.fetchMessagePart(UID, path)
}
