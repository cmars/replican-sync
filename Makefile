
ARCH=6

G=$(HOME)/go/bin/$(ARCH)g
L=$(HOME)/go/bin/$(ARCH)l

fibo:	fibo.6
	$(L) -o $@ $^

%.6:	%.go
	$(G) -o $@ $<

clean:
	$(RM) *.6 fibo


