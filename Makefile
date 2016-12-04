test:
	go test -v

get:
	go get github.com/miekg/dns
	go get github.com/akira/go-puppetdb
	go get gopkg.in/yaml.v2

build:
	go build puppetdb-dns.go

docker-build:
	docker build -t jfroche/puppetdb-dns .

docker-run:
	docker run -p 5354 -v $(PWD)/dns.conf:/go/dns.conf jfroche/puppetdb-dns

run:
	./puppetdb-dns

.PHONY: release

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build -o release/puppetdb-dns-linux-amd64 github.com/jfroche/puppetdb-dns
	GOOS=linux GOARCH=386 go build -o release/puppetdb-dns-linux-386 github.com/jfroche/puppetdb-dns
