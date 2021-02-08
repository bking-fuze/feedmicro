#EXECUTABLES=fetch server
EXECUTABLES=server

all: $(EXECUTABLES)

fetch: fetch.go getlogs.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ $^

server: server.go handlers.go httputil.go getlogs.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ $^

clean:
	rm -f $(EXECUTABLES)
