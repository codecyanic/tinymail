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

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/codecyanic/tinymail/internal/clients/imap"
)

const defaultAddr = ":9009"

func usage() {
	fmt.Printf(`Usage: tinymail [[<address>]:<port>]
Start tinymail mail client web app.

Arguments:
  [[<address>]:]<port>  optional listen address and port (default: %s)

Options:
  -h, --help            display this help and exit
`, defaultAddr)
}

func registerHandlers(static fs.FS) {
	http.Handle("/", http.FileServer(http.FS(static)))

	http.HandleFunc("/api/account", func(w http.ResponseWriter, r *http.Request) {
		email, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		client := imap.NewClient(email, password)

		w.Header().Set("Content-Type", "application/json")
		account, err := client.Account()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("%s: %v\n", r.URL.Path, err)
			return
		}

		json.NewEncoder(w).Encode(account)
	})

	http.HandleFunc("/api/mailbox/{mailbox}", func(w http.ResponseWriter, r *http.Request) {
		email, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		client := imap.NewClient(email, password)
		w.Header().Set("Content-Type", "application/json")

		mailboxName := r.PathValue("mailbox")
		mailbox, err := client.Mailbox(mailboxName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("%s: %v\n", r.URL.Path, err)
			return
		}

		json.NewEncoder(w).Encode(mailbox)
	})

	http.HandleFunc("/api/mailbox/{mailbox}/messages/{uids}", func(w http.ResponseWriter, r *http.Request) {
		email, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mailbox := r.PathValue("mailbox")
		var uids []uint32
		for _, uid := range strings.Split(r.PathValue("uids"), ",") {
			n, err := strconv.ParseUint(uid, 10, 32)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			uids = append(uids, uint32(n))
		}

		client := imap.NewClient(email, password)
		w.Header().Set("Content-Type", "application/json")

		messages, err := client.Messages(mailbox, uids)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("%s: %v\n", r.URL.Path, err)
			return
		}

		json.NewEncoder(w).Encode(messages)
	})

	http.HandleFunc("/api/mailbox/{mailbox}/message/{uid}", func(w http.ResponseWriter, r *http.Request) {
		email, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mailbox := r.PathValue("mailbox")
		uid, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		client := imap.NewClient(email, password)
		w.Header().Set("Content-Type", "application/json")

		message, err := client.Message(mailbox, uint32(uid))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("%s: %v\n", r.URL.Path, err)
			return
		}

		json.NewEncoder(w).Encode(message)
	})
}

//go:embed static/index.html
//go:embed static/app.js
//go:embed static/style.css
var content embed.FS

func main() {
	args := os.Args[1:]
	help := len(args) == 1 && (args[0] == "-h" || args[0] == "--help")
	if len(args) > 1 || help {
		usage()
		return
	}

	addr := defaultAddr
	if len(args) == 1 {
		addr = args[0]
	}

	static, err := fs.Sub(content, "static")
	if err != nil {
		panic(err)
	}

	registerHandlers(static)

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
