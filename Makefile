build:
		go build -o bin/fs

run: build
		./bin/fs

test:
		#go test ./... -v
	    go test ./p2p -v