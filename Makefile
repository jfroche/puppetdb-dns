include .env

get:
	go get github.com/miekg/dns
	go get github.com/akira/go-puppetdb
	go get gopkg.in/yaml.v2

test:
	go test

build:
	go build puppetdb-dns.go

docker-build:
	docker build -t jfroche/puppetdb-dns .

docker-run:
	docker run -p 5354 -v $(PWD)/dns.conf:/go/dns.conf jfroche/puppetdb-dns

run:
	./puppetdb-dns
