# Metal-Core

Metal-Core provides a REST API
1. as an endpoint for Pixiecore API implementation to enable custom PXE boot and
1. as a Metal API middleware

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

