System design
=============

Interface Database separates SQL data access from logic

file structs.go contains common data structures

handling of received messages is done by pipelining so it's
easy to change / inject middleware like virus checking

