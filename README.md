# TinyMail

TinyMail is a minimalistic, self-hostable webmail client designed for
simplicity and security. It offers a lightweight alternative to heavier
PHP-based email clients.

## Features

- Single binary server, under 10MB in size and memory use
- Automatic configuration via SRV records
- Simple web interface
- Plain text and UTF-8 email support
- Only secure IMAP/SMTP protocols (unencrypted and STARTTLS not supported)
- Does not store emails on the server
- Free and open-source software (AGPLv3)

## Tech Stack

Backend is written in Go and frontend in vanilla JavaScript.
The project uses a small number of Go libraries:
    [go-imap](https://github.com/emersion/go-imap),
    [go-smtp](https://github.com/emersion/go-smtp),
    [letters](https://github.com/mnako/letters).

## Build and Run

### Requirements

For building the release binary Docker, Git and Make are required.

### Build

```bash
git clone https://github.com/codecyanic/tinymail.git
cd tinymail
make build
```

This will produce a standalone `tinymail` binary.

### Run

```bash
./tinymail
```

By default, the server listens on `:9009`. To bind to a different address:

```bash
./tinymail 127.0.0.1:8000
```

### Usage

Open your browser at:

```
http://localhost:9009
```

You can use the interface to log into your email account using your email and
password. Keep in mind that your email's domain's DNS needs to have valid SRV
records for automatic configuration of IMAP and SMTP, and the servers need to
support strict TLS and PLAIN authentication.

## Deployment

### systemd Service Example

Create a file at `/etc/systemd/system/tinymail.service`:

```ini
[Unit]
Description=TinyMail Email Client
After=network.target
Requires=network.target

[Service]
Type=simple
DynamicUser=yes
ExecStart=/opt/tinymail/tinymail 127.0.0.1:8000
WorkingDirectory=/opt/tinymail
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reexec
sudo systemctl enable --now tinymail
```

## License

TinyMail is licensed under the [GNU AGPLv3](LICENSE).

The server automatically bundles and distributes the source code for easier
compliance.

## Roadmap

This project was developed in under two weeks and is currently minimal. Planned
improvements include:

- Support for IDLE (push notifications)
- Ability to reply to emails
- Ability to specify allowed domains/hosts

## Acknowledgments

Thanks to [Simon Ser](https://emersion.fr/) for the IMAP and SMTP libraries
used in this project.
