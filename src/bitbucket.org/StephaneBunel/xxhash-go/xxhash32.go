//   Copyright (C) 2013, Stéphane Bunel
//   BSD 2-Clause License (http://www.opensource.org/licenses/bsd-license.php)

// Packages xxHash is a wrapper for C xxHash - an extremely fast Hash algorithm,
// working at speeds close to RAM limits.
package xxhash

/*
#include "C/xxhash.c"
*/
import "C"

import (
	"errors"
	"hash"
	"unsafe"
)

// Digest32 represents the partial evaluation of a checksum.
type Digest32 struct {
	seed   uint32         // Memorize seed for Reset()
	result uint32         // Result of XXH32_result()
	state  unsafe.Pointer // (void*)struct XXH_state32_t
}

// New32Seed returns a new hash.Hash32 computing the xxhash32 checksum with a seed.
func New32Seed(seed uint32) hash.Hash32 {
	d := new(Digest32)
	d.seed = seed
	d.state = C.XXH32_init(C.uint(seed))
	return d
}

// New32 returns a new hash.Hash32 computing the xxHash32 checksum.
func New32() hash.Hash32 {
	return New32Seed(0)
}

// Size returns the number of bytes Sum will return.
func (self *Digest32) Size() int {
	return 4
}

// Reset resets the hash to one with zero bytes written.
func (self *Digest32) Reset() {
	if self.state != nil {
		C.XXH32_result(self.state)
	}
	self.state = C.XXH32_init(C.uint(self.seed))
}

func (self *Digest32) BlockSize() int {
	return 1
}

// Write adds more data to the running hash.
// Length of data MUST BE less than 1 Gigabytes.
func (self *Digest32) Write(data []byte) (nn int, err error) {
	if self.state == nil {
		return 0, errors.New("Cannot add more data to already computed checksum.")
	}

	l := len(data)
	if l > 1<<30 {
		return 0, errors.New("Cannot add more than 1 Gigabytes at once.")
	}
	C.XXH32_feed(self.state, unsafe.Pointer(&data[0]), C.int(l))
	return len(data), nil
}

func (self *Digest32) Sum(in []byte) []byte {
	h := self.Sum32()
	in = append(in, byte(h>>24))
	in = append(in, byte(h>>16))
	in = append(in, byte(h>>8))
	in = append(in, byte(h))
	return in
}

// Sum32 returns xxhash32 checksum and free internal state.
func (self *Digest32) Sum32() uint32 {
	if self.state != nil {
		self.result = uint32(C.XXH32_result(self.state))
		self.state = nil
	}
	return self.result
}

// Checksum32 returns the xxhash32 checksum of data. Length of data MUST BE less than 2 Gigabytes.
func Checksum32(data []byte) uint32 {
	return uint32(C.XXH32(unsafe.Pointer(&data[0]), C.int(len(data)), C.uint(0)))
}

// Checksum32Seed returns the xxhash32 checksum of data using a seed. Length of data MUST BE less than 2 Gigabytes.
func Checksum32Seed(data []byte, seed uint32) uint32 {
	return uint32(C.XXH32(unsafe.Pointer(&data[0]), C.int(len(data)), C.uint(seed)))
}
