Henrymail
=========
This project is intended to build an all-in-one email 
platform that's easy to configure, easy to administer,
reasonably reliable, and secure by default.

The project is not intended to be extremely scalable,
or to suit enterprise users. Individual components (e.g. 
the mail transfer agent) are not replaceable with other 
software. This is an all in-in-one package only. 

Usage
=====
* Prerequisites: You should know how to obtain an internet 
facing server, and open up ports 25, 143, 443 & 587. 
You also need a domain name, and you need to be able to 
configure DNS records for it.
* A raspberry PI is more than capable of running this 
software.
* This guide will be mainly focused on unix-like operating
systems.  
* Clone the source code and build the application 
```bash
go get gogs.mfashby.net/martin/henrymail
cd $GOPATH/src/gogs.mfashby.net/martin/henrymail
go build 
```
* Copy the sample configuration and amend it to meet your 
needs
```bash
cp henrymail.sample.prop henrymail.prop
```
* `Domain` should be something like `example.com`. 
Email addresses on this server will then be like
`admin@example.com` 
* `ServerName` should be the domain name for the mail 
server itself, e.g. `mail.example.com`
* `AutoCertEmail` is an address where information about 
TLS certificates will be sent to (e.g. if your certificate
is about to expire). Set this to your normal email address.
* `AdminUserName` is the address you would like to create 
as the site administrator. This user will be created
automatically when the application starts for the first 
time.
* Create a new user to run the application (it's better 
for security than running the application as root)
* Ensure that the new user has permission to bind to low 
port numbers.
* Open up the relevant ports on the firewall.
* Set up an A record DNS entry for your email server 
`EXAMPLE REQUIRED`. 
* Set up an MX record DNS entry for your domain `EXAMPLE REQUIRED`
* Start the application
```bash
./henrymail
```
* Note administrator user's password which is generated and 
logged to the console.
* Open a web browser and navigate to your mail server's 
address
* Log in with the administrator email and password. 
* Configure DKIM and SPF records as directed on the 
health-checks page.
* Add users.
* You can log in with your favourite email client 
(e.g. Thunderbird)
* Or you can use the built-in webmail client. 