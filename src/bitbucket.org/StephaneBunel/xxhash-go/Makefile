SOURCES  := $(wildcard *.go xxh32sum/*.go)

ALL: README.md $(SOURCES)
	go test -ldflags "-s"
	go install bitbucket.org/StephaneBunel/xxhash-go/xxh32sum

README.md: .godocdown.md
	godocdown -plain=true >README.md
