Goals of the project
====================
* Complete email server (SMTP, IMAP, Webmail, Admin interface)
* Works with common email clients
* Single statically compiled binary for super easy deployment
* Minimal hardware requirements (CPU & Memory)
* Multi-platform
* Configurable with sensible default configuration
* Ideally configurable via env vars, cmd line flags, or config file.
* Extensible 
* Modifiable
* Nice docs for setting it up
* Ideally all unit / integration / e2e tested

How to do it
============
* Emersion SMTP https://github.com/emersion/go-smtp
* Emersion IMAP https://github.com/emersion/go-imap
* Emersion Message https://github.com/emersion/go-message
* SQLite 3 storage https://github.com/mattn/go-sqlite3
* HTML templating https://golang.org/pkg/html/template/
* Config github.com/spf13/viper
* Auto cert https://godoc.org/golang.org/x/crypto/acme/autocert
* DKIM

Additional really useful features
=================================
* Spam filtering https://github.com/BlogSpam-Net/blogspam-api
* Antivirus https://github.com/dutchcoders/go-clamd
* DoS / brute force protection
* Probably read https://blog.cloudflare.com/exposing-go-on-the-internet/
* CSRF protection
* SQLite concurrency control
* Sending retry with https://github.com/robfig/cron

Order to do it in
=================
* Get a basic user admin interface
* Get working SMTP & IMAP implementation backed by a database
* Fill in all the remaining features as you like.
* Try to keep it modular