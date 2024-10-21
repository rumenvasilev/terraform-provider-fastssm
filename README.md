# Terraform Provider: FastSSM

The FastSSM provider was written to speed-up terraform runs across larger installations/usage, where AWS SSM parameter is employed. And so it handles only the lifecycle of AWS SSM parameter (both resource and data-source), but much faster. There's a limited set of attributes supported (omitted some of the metadata) in order to reduce the amount of expensive (latency, rate-limit) API calls, sometimes by more than 95%. There are no plans to extend it with support for any other AWS resources.

The ssm_parameter resource can be used to create a AWS SSM [parameter](docs/resources/parameter.md).

**THIS PROVIDER IS STILL EXPERIMENTAL AND IN ACTIVE DEVELOPMENT! USE AT YOUR OWN RISK!**

## Why write a separate provider?

Great question! Initial plan was to follow-up with some community ideas in terraform-provider-aws to improve the performance. However after reviewing the codebase (that's one of the first terraform proviers ever), I decided it would be much faster and easier to just write just this resource from scratch in a provider envelope, without the difficulty of maintaining full backward compatibility with every feature terraform-provider-aws had.

Starting fresh allowed the full adoption of the latest terraform-provider-framework, making it a lot simpler and easier to bootstrap a POC.

My tests so far have proven read operations (the aim of this optimization), with thousands of SSM parameters, have gone down from multiple minutes, down to seconds. And thus terraform plan (refresh) and terraform destroy commands have both benefitted greatly. The refresh is the most abused operation with existing systems.

## Documentation, questions and discussions

Official documentation on how to use this provider can be found on the
[Terraform Registry](https://registry.terraform.io/providers/rumenvasilev/fastssm/latest/docs).
In case of specific questions or discussions, please use the
[GitHub issues here](https://github.com/rumenvasilev/terraform-provider-fastssm/issues).

We also provide:

* [Support](.github/SUPPORT.md) page for help when using the provider
* [Contributing](.github/CONTRIBUTING.md) guidelines in case you want to help this project

The remainder of this document will focus on the development aspects of the provider.

## Requirements

* [Terraform](https://www.terraform.io/downloads) (>= 0.12)
* [Go](https://go.dev/doc/install) (1.23)
* [GNU Make](https://www.gnu.org/software/make/)
* [golangci-lint](https://golangci-lint.run/usage/install/#local-installation) (optional)

## Development

### Building

1. `git clone` this repository and `cd` into its directory
2. `make build` will trigger the Golang build

The provided `GNUmakefile` defines additional commands generally useful during development,
like for running tests, generating documentation, code formatting and linting.
Taking a look at it's content is recommended.

### Testing

In order to test the provider, you can run

* `make test` to run provider tests
* `make testacc` to run provider acceptance tests

It's important to note that acceptance tests (`testacc`) will actually spawn
`terraform` and the provider. Read more about they work on the
[official page](https://www.terraform.io/plugin/sdkv2/testing/acceptance-tests).

### Generating documentation

This provider uses [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs/)
to generate documentation and store it in the `docs/` directory.
Once a release is cut, the Terraform Registry will download the documentation from `docs/`
and associate it with the release version. Read more about how this works on the
[official page](https://www.terraform.io/registry/providers/docs).

Use `make generate` to ensure the documentation is regenerated with any changes.

### Using a development build

If [running tests and acceptance tests](#testing) isn't enough, it's possible to set up a local terraform configuration
to use a development builds of the provider. This can be achieved by leveraging the Terraform CLI
[configuration file development overrides](https://www.terraform.io/cli/config/config-file#development-overrides-for-provider-developers).

First, use `make install` to place a fresh development build of the provider in your [`${GOBIN}`](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies) (defaults to `${GOPATH}/bin` or `${HOME}/go/bin` if `${GOPATH}` is not set). Repeat
this every time you make changes to the provider locally.

Then, in your `${HOME}/.terraformrc` (Unix) / `%APPDATA%\terraform.rc` (Windows), a `provider_installation` that contains
the following `dev_overrides`:

```hcl
provider_installation {
  dev_overrides {
    "rumenvasilev/fastssm" = "${GOBIN}" //< replace `${GOBIN}` with the actual path on your system
  }

  direct {}
}
```

Note that it's also possible to use a dedicated Terraform configuration file and invoke `terraform` while setting
the environment variable `TF_CLI_CONFIG_FILE=my_terraform_config_file`.

Once the `dev_overrides` are in place, any local execution of `terraform plan` and `terraform apply` will
use the version of the provider found in the given `${GOBIN}` directory,
instead of the one indicated in your terraform configuration.

### Testing GitHub Actions

This project uses [GitHub Actions](https://docs.github.com/en/actions/automating-builds-and-tests) to realize its CI.

Sometimes it might be helpful to locally reproduce the behaviour of those actions,
and for this we use [act](https://github.com/nektos/act). Once installed, you can _simulate_ the actions executed
when opening a PR with:

```shell
# List of workflows for the 'pull_request' action
$ act -l pull_request

# Execute the workflows associated with the `pull_request' action 
$ act pull_request
```

## Releasing

The release process is automated via GitHub Actions, and it's defined in the Workflow
[release.yml](./.github/workflows/release.yml).

Each release is cut by pushing a [semantically versioned](https://semver.org/) tag to the default branch.

## License

[Apache 2.0](./LICENSE)