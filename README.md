# metal-core

metal-core dynamically reconfigures switches based on the state held in the metal-api. Therefore, it must run on every leaf switch and have control over the configuration files for network interfaces and the routing suite (`/etc/frr/frr.config`) of the switches.

In the PXE-boot process of machines `metal-core` will act as a proxy between API-requests issued by `pixiecore` and the `metal-api`. The `metal-api` will answer with a mini OS (see [metal-hammer](https://github.com/metal-stack/metal-hammer) and [kernel](https://github.com/metal-stack/kernel)).

Besides that, it ensures the proper boot order (IPMI) and monitors their liveliness with [LLDP](https://github.com/metal-stack/go-lldpd)).

## Building

### Install mage

https://magefile.org/

Set GOPATH to a writeable directory of your choice.

```
go get -u -d github.com/magefile/mage
cd $GOPATH/src/github.com/magefile/mage
go run bootstrap.go
```

The "mage" binary will be placed in $GOPATH/bin, add this to your PATH.

### Run mage

```
mage build
```

This triggers
* the installation of build dependencies like swagger
* the generation of the client code for spec/metal-core.json and domain/metal-api.json
* after that, bin/metal-core is built

