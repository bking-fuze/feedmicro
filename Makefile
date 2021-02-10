EXECUTABLES=server

all: $(EXECUTABLES)

server: server.go httputil.go logsget.go db.go aws.go logspost.go loguploadurl.go auth.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ $^

clean:
	rm -f $(EXECUTABLES)
