# Azion Terraform Provider

[![MIT License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/aziontech/terraform-provider-azion)](https://goreportcard.com/report/github.com/aziontech/terraform-provider-azion)
<!--- # Documentation: https://registry.terraform.io/providers/azion/azion/latest/docs
-->

## Quick links
* [Requirements](#Requirements)
* [Building](#building)
* [Testing](#Testing)
* [Contributing](CONTRIBUTING.md)
* [Code of Conduct](CODE_OF_CONDUCT.md)
* [License](#License)

## Requirements
-	[Terraform](https://www.terraform.io/downloads.html) 1.4.x or higher
-	[Go](https://golang.org/doc/install) 1.24+ (to build the provider plugin)


## Building
To build or extends the Azion Terraform Provider, you'll first need [Go](http://www.golang.org)
installed on your machine (version 1.24+ is _required_). You'll also need to
correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well
as adding `$GOPATH/bin` to your `$PATH`.


Clone repository to: `$GOPATH/src/github.com/aziontech/terraform-provider-azion`

```sh
$ mkdir -p $GOPATH/src/github.com/aziontech; cd $GOPATH/src/github.com/aziontech
$ git clone git@github.com:aziontech/terraform-provider-azion
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/aziontech/terraform-provider-azion
$ make build
```

## Testing

See above for which option suits your workflow for building the provider.

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

_Note:_ Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```

To run a subset of the acceptance test suite, you can run

```sh
TESTARGS='-run "^<regex target of tests>" -count 1 -parallel 1' make testacc
```

*Note:* Acceptance tests create real resources, and often cost money to run. You should expect that the full acceptance test suite will take hours to run.


## License
This project is licensed under the terms of the [MIT](LICENSE) license.



