
# replican-sync - An rsync algorithm implementation in Go #

## Features ##

replican-sync currently supports local file & directory synchronization with the rsync algorithm.

### Implemented ###

* *nix platforms.
* Hierarchical, content-addressable filesystem model down to the block level.
* Match and patch files with rolling checksum and strong cryptographic hash.
* Match and patch directory structures.

### Planned/In Development ###

* Directory structure patching on MinGW.
* Handle symbolic links.
* Synchronization behavior options (filtering, handling deletes, etc.)
* Performance benchmarking, tuning, optimization.

## Why

I'm working on a folder synchronization service/application. replican-sync is just the 
first step.

## License

MIT, see LICENSE. If you use replican-sync, I'd like to hear from you.

## Developers

replican is developed with Go.

### Building

You'll need to goinstall:

* github.com/bmizerany/assert
* optarg.googlecode.com/hg/optarg

Run 'gb' from the src subdirectory. 

### Testing

I developed a utility to easily fabricate directory structures for testing directory 
indexing, matching & patching. See replican/treegen.

