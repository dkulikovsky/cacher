# xxhash-go

xxhash-go is a go (golang) wrapper for C [xxhash](http://code.google.com/p/xxhash/) - an extremely fast Hash algorithm,
working at speeds close to RAM limits.

xxhash-go provides package (xxhash) for go developers and one command line utility: xxh32sum.

## Install

Assuming that [go1](http://code.google.com/p/go/downloads/list) and [mercurial](http://mercurial.selenic.com/wiki/Download) are installed.

    go get bitbucket.org/StephaneBunel/xxhash-go
    go install bitbucket.org/StephaneBunel/xxhash-go/xxh32sum

## Benchmark

xxHash_test.go includes a quick and dirty benchmark.

```go test bitbucket.org/StephaneBunel/xxhash-go -bench=".*"```

Core i5-3570K CPU @ 3.40GHz, x86_64 GNU/Linux 3.5.0:

```
Benchmark_xxhash32           50000000       43.2 ns/op (C binding)
Benchmark_goxxhash32         50000000       66.4 ns/op (Pure Go)
Benchmark_CRC32IEEE          10000000      149.0 ns/op
Benchmark_Adler32            20000000       90.2 ns/op
Benchmark_Fnv32              10000000      154.0 ns/op
Benchmark_MurmurHash3Hash32    500000     3080.0 ns/op
```

xxhash32 is more than two time faster than it's best competitor !

## xxh32sum

Usage: (Assuming that $GOPATH/bin is in your $PATH)

    % xxh32sum -h
    Usage: xxh32sum [<OPTIONS>] <filename> [<filename>] [...]
    OPTIONS:
      -readsize=1048576: Read buffer size
      -version=false: Show version

Checksum a file

    % xxh32sum /etc/passwd
    d2582536    /etc/passwd

Checksum from stdin

    % cat /etc/passwd | xxh32sum -
    d2582536

## xxHash package

### Examples

```
import xxh "bitbucket.org/StephaneBunel/xxhash-go"

h32 := xxh.Checksum32([]byte("Lorem ipsum..."))
```

See [xxhash32_test.go](https://bitbucket.org/StephaneBunel/xxhash-go/src/tip/xxhash32_test.go?at=default) and
[xxh32sum](https://bitbucket.org/StephaneBunel/xxhash-go/src/tip/xxh32sum/main.go?at=default) source code.

## Usage

## Usage

#### func  Checksum32

    func Checksum32(data []byte) uint32

Checksum32 returns the xxhash32 checksum of data. Length of data MUST BE less
than 2 Gigabytes.

#### func  Checksum32Seed

    func Checksum32Seed(data []byte, seed uint32) uint32

Checksum32Seed returns the xxhash32 checksum of data using a seed. Length of
data MUST BE less than 2 Gigabytes.

#### func  GoChecksum32

    func GoChecksum32(data []byte) uint32

Pure Go implementation of Checksum32()

#### func  GoChecksum32Seed

    func GoChecksum32Seed(data []byte, seed uint32) uint32

Pure Go implementation of Checksum32Seed()

#### func  GoNew32

    func GoNew32() hash.Hash32

Pure Go implementation of New32()

#### func  GoNew32Seed

    func GoNew32Seed(seed uint32) hash.Hash32

Pure Go implementation of GoNew32Seed()

#### func  New32

    func New32() hash.Hash32

New32 returns a new hash.Hash32 computing the xxHash32 checksum.

#### func  New32Seed

    func New32Seed(seed uint32) hash.Hash32

New32Seed returns a new hash.Hash32 computing the xxhash32 checksum with a seed.

#### type Digest32

    type Digest32 struct {
    }


Digest32 represents the partial evaluation of a checksum.

#### func (*Digest32) BlockSize

    func (self *Digest32) BlockSize() int


#### func (*Digest32) Reset

    func (self *Digest32) Reset()

Reset resets the hash to one with zero bytes written.

#### func (*Digest32) Size

    func (self *Digest32) Size() int

Size returns the number of bytes Sum will return.

#### func (*Digest32) Sum

    func (self *Digest32) Sum(in []byte) []byte


#### func (*Digest32) Sum32

    func (self *Digest32) Sum32() uint32

Sum32 returns xxhash32 checksum and free internal state.

#### func (*Digest32) Write

    func (self *Digest32) Write(data []byte) (nn int, err error)

Write adds more data to the running hash. Length of data MUST BE less than 1
Gigabytes.

## License

[BSD 2-Clause License][bsd-licence]

Copyright (c) 2013, Stéphane Bunel (@StephaneBunel)  
All rights reserved.

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

- Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

---

Doc produced by [godocdowm][]: ```godocdown --plain=true >README.md```


[bsd-licence]:  http://opensource.org/licenses/bsd-license.php
[godocdowm]:    https://github.com/robertkrimen/godocdown
