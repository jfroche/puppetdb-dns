# PuppetDB-dns

Query puppet [facts](https://docs.puppet.com/facter) from [PuppetDB](https://docs.puppet.com/puppetdb/)
through a DNS interface.

We use the [role/profile pattern](https://docs.puppet.com/pe/2016.4/r_n_p_intro.html). Each machines have a role.
As our infrastructure become more and more complex we would like to quickly lookup nodes that match criteria.
E.g. give me the servers with the `app1` role in datacenter `brussels` should be as simple as:

```
# host app1.brussels.puppetdb
app1.brussels.puppetdb has address 192.168.99.1
```

But you might want as well query nodes using other facts. That's why we made a configuration file that list the combinations
of facts that can be used together:

```yaml
domain: puppetdb
bind: 127.0.0.1
port: 5353
ttl: 86400
verbose: true
puppetdb: http://puppetdb.prd.srv.mycompany.be:8080
hierarchy:
  - [role, datacenter]
  - [subgroup, zone]
  - [subgroup, role, zone]
  - [subgroup, role, zone, hostgroup]
```

Every part of the subdomain will be mapped to each element of the hierachy. So if we query `app1.brussels.puppetdb`, with the previous confiugration file, there will be two queries on puppetdb:

 1. `["and", ["=", ["fact", "role"], "app1"], ["=", ["fact", "datacenter"], "brussels"]]`
 2. `["and", ["=", ["fact", "subgroup"], "app1"], ["=", ["fact", "zone"], "brussels"]]`

## Installation

The easiest way to install is to run:

```shell
$ go get github.com/jfroche/puppetdb-dns
$ go install github.com/jfroche/puppetdb-dns
```

## Usage

To run `puppetdb-dns` with a configuration file:

```shell
$ puppetdb-dns -conf dns.conf
```

To run `puppetdb-dns` with `dns.conf` as default configuration file:

```shell
$ puppetdb-dns
```

You can query your local `puppetdb-dns` using dig:

```shell
$ dig @localhost -p 5353 novac.site.prod.plone.cicd
```

## Configuration file

The yaml configuration file should contain these keys:

 - domain (text): the domain that the dns server answer to
 - bind (text): network ip the dns server listen to
 - port (int): port the dns server listen to
 - ttl (int): time to live of the resource records
 - verbose (bool): print debug information
 - puppetdb (text): URL of the puppetdb to query
 - hierarchy (list of list of string): contains the list of list of facts that will be mapped against dns query

## Docker

`puppetdb-dns` can be run using docker.

Create your dns.conf file and run:

```shell
$ docker run -p 53:53 -v $(pwd)/dns.conf:/go/dns.conf jfroche/puppetdb-dns
```

### Based on:

 * [microdns](https://github.com/fffaraz/microdns.git) - Basic DNS server in go
