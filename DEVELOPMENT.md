# Development
## Tools required: 
* [Go 1.11 or above](https://golang.org/dl/) 
* [xo/xo](https://github.com/xo/xo#installation) for generating database boilerplate
* [aprice/embed](https://github.com/aprice/embed#installation) for embedding files into the binary

## Building & running
Clone the repository 
```bash
git clone https://github.com/MFAshby/henrymail.git
```

Add a configuration file. The sample configuration file for local development should work
well for local development
```bash
cd henrymail
cp henrymail.dev.prop henrymail.prop
```

Build & run the application
```bash
go generate ./...
go build -o dist/henrymail
dist/henrymail
```

The interfaces started will be listed on standard output. 
With the development configuration, emails sent to example.com 
should loop straight back to ourselves. 

## Contributing
Please first read the [code of conduct](./CODE_OF_CONDUCT.md)

Outstanding bugs and feature requests are tracked via [GitHub issues](https://github.com/MFAshby/henrymail/issues),
if you have something new you'd like to add please also add an issue for discussion. 

Pull requests are welcome and appreciated, and I'll try to review them in a timely fashion.


Please bear in mind this is currently a hobby level project, and is not yet suitable for 
enterprises or critical situations.  

## Technical overview
henrymail is designed to be a single binary, all-in-one mail server. It runs several TCP
servers for email interfaces, and an http(s) interface for serving administration pages. 
Data is stored in an sqlite database. Mail processing happens via a pipeline type architecture, 
in which each element of the pipeline (e.g. virus scanning, signature checking) is unaware of 
the others. The top level main.go file assembles the pipeline and starts the servers.