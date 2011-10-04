GOROOT=$(HOME)/go

include $(GOROOT)/src/Make.inc

TARG=replican
GOFILES=\
	blocks.go\

include $(GOROOT)/src/Make.pkg

