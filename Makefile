GOPATH ?= /usr/share/gocode

build: leader-elect

leader-elect: leader-elect.go
	go build leader-elect.go

