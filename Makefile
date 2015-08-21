GOPATH ?= /usr/share/gocode

build: master-elect

master-elect: master-elect.go
	go build master-elect.go

