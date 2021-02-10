EXECUTABLES=server

all: $(EXECUTABLES)

server: server.go handlers.go httputil.go getlogs.go db.go aws.go putlog.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ $^

clean:
	rm -f $(EXECUTABLES)
