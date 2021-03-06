
# replican-sync - Filesystem synchronization for Go #

## Features ##

replican-sync provides local file & directory synchronization with an implementation of the [rsync algorithm](http://rsync.samba.org/tech_report/). 
It is not compatible with the wire protocols and indexing used in the [rsync(1)](http://www.samba.org/ftp/rsync/rsync.html) utility.

## Status ##

At this point, I'm working self-consistency and simplicity into the library 
as I develop on it. The API is subject to change.

### Implemented ###

* Linux, OSX, MinGW
* Hierarchical, [content-addressable](http://en.wikipedia.org/wiki/Content-addressable_storage) filesystem model down to the block level.
* Match and patch files with rolling checksum and strong cryptographic hash.
* Match and patch directory structures.

### Planned/In Development ###

In order of current precedence.

* (NEW) Simple version tracking and automated merge between stores.
  * Emphasis on the 'simple'! This isn't going to be a DVCS! :)
* Synchronization behavior options (filtering, handling deletes, etc.)
* Handle symbolic links.
* Performance benchmarking, tuning, optimization.

## Getting Started

	goinstall github.com/cmars/replican-sync/replican/sync

See rp.go, fs\_test.go and merge\_test.go for examples.

## Why?

I'm working on a decentralized folder synchronization service/application. 
replican-sync is just the first step.

## License

MIT, see LICENSE. If you use replican-sync, I'd like to hear from you.

## Developers

replican is developed in [Go](http://golang.org/).

### Building

You'll need to first goinstall:

* github.com/bmizerany/assert
* optarg.googlecode.com/hg/optarg

Run [gb](https://github.com/skelterjohn/go-gb) from the top level. 

### Testing

'gb -t' to execute unit tests.

Indexing, matching & patching are tested with a little utility that 
fabricates directory structures of arbitrary random, but reproducible binary data.
See replican/treegen.

