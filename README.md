
# replican-sync - An rsync algorithm implementation in Go #

## Features ##

replican-sync currently supports local file & directory synchronization with the [rsync algorithm](http://rsync.samba.org/tech_report/).

### Implemented ###

* *nix platforms.
* Hierarchical, [content-addressable](http://en.wikipedia.org/wiki/Content-addressable_storage) filesystem model down to the block level.
* Match and patch files with rolling checksum and strong cryptographic hash.
* Match and patch directory structures.

### Planned/In Development ###

* Directory structure patching on MinGW.
* Handle symbolic links.
* Synchronization behavior options (filtering, handling deletes, etc.)
* Performance benchmarking, tuning, optimization.

## Why?

I'm working on a decentralized folder synchronization service/application. 
replican-sync is just the first step.

## License

MIT, see LICENSE. If you use replican-sync, I'd like to hear from you.

## Developers

replican is developed in [Go](http://golang.org/).

### Building

You'll need to goinstall:

* github.com/bmizerany/assert
* optarg.googlecode.com/hg/optarg

Run [gb](https://github.com/skelterjohn/go-gb) from the src subdirectory. 

### Testing

'gb -t' to execute unit tests.

Indexing, matching & patching are tested with a little utility that 
fabricates directory structures of arbitrary random, but reproducible binary data.
See replican/treegen.

