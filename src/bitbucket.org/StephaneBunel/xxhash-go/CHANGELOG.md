# CHANGELOG
xxhash-go

## 2013.03.10-1
- Include a pure Go implementation of xxhash by vova16.
  Pure Go API is prefixed by "Go". See xxhash32_test.go
- Update benchmark to include pure Go implementation.

## 2013.03.08-1
- API changed. It Now implements Hash.Hash32 interface.

## 2013.03.08-2
- Add some benchmark functions in xxhash_test.go. No surprise xxhash is the winner.

## 2013.02.08-1
- Add options to xxh32sum command (--readsize, --version).

## 2013.02.07-1
- Writing wrapper and tests
