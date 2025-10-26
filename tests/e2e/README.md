# E2E Tests for terraform-provider-fastssm

This directory contains end-to-end (e2e) tests for the FastSSM Terraform provider using LocalStack to simulate AWS SSM.

## Overview

The e2e tests validate the provider's functionality against a real (simulated) SSM service, testing:

- ✅ **CRUD Operations**: Create, Read, Update, Delete SSM parameters
- ✅ **Data Sources**: Reading parameters using data sources
- ✅ **Import**: Importing existing SSM parameters into Terraform state
- ✅ **Updates**: Updating parameter values and verifying version increments
- ✅ **Multiple Types**: String, StringList, and SecureString parameter types
- ✅ **Advanced Features**: Descriptions, allowed patterns, data types

## Prerequisites

### Local Testing

1. **Docker & Docker Compose**: Required to run LocalStack
2. **Go 1.25+**: To build the provider (optional - script will build if not already built)
3. **Terraform 1.10+**: Required for running tests
4. **AWS CLI** or **awslocal**: For direct SSM interactions in tests (awslocal preferred)
5. **jq**: For JSON parsing in test scripts

> **Note**: The test script will automatically build the provider if it's not already installed. In CI/CD environments where the provider is pre-built, the build step is skipped for efficiency.

### Provider Development Override

The tests use Terraform's `dev_overrides` feature to ensure the local development version is used instead of pulling from the public registry. The test script automatically:
1. Builds the provider to `~/.local/share/terraform/plugins/`
2. Creates `e2e-test.tfrc` in the test directory with `dev_overrides` configuration
3. Sets `TF_CLI_CONFIG_FILE` environment variable to point to this config file
4. Cleans up the config file after tests complete

Per [Terraform's CLI configuration documentation](https://developer.hashicorp.com/terraform/cli/config/config-file#locations), the `TF_CLI_CONFIG_FILE` environment variable allows using a custom config file (must end with `.tfrc` or `.terraformrc`), so your existing `~/.terraformrc` is never touched.

Test configurations specify `version = ">= 0.1.0"` which is satisfied by any version, but the `dev_overrides` ensures the local build is used instead of fetching from the registry.

**Expected behavior:**
- Terraform will show a warning that dev overrides are active (this is correct!)
- Your existing `~/.terraformrc` is completely untouched
- Your local code changes are always tested, not published registry versions
- Terraform update checks are disabled (`CHECKPOINT_DISABLE=1`)

### Installation

```bash
# On macOS
brew install go terraform awscli jq docker

# On Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y golang terraform awscli jq docker.io docker-compose

# On RHEL/CentOS
sudo yum install -y golang terraform awscli jq docker docker-compose
```

## Running Tests Locally

The repository includes a `docker-compose.yml` that sets up LocalStack identically to the GitHub Actions CI environment.

**Enhanced Healthcheck:** The Docker Compose configuration includes a two-stage healthcheck that:
1. Verifies SSM service status is "running" in LocalStack
2. Actually calls the SSM API (`describe-parameters`) to ensure it's functional and activate the lazy-loaded service

This ensures LocalStack is fully ready before tests begin.

### Option 1: Using Make (Recommended)

```bash
# From the repository root

# Run full e2e test suite (starts LocalStack, runs tests, cleans up)
make e2e-test

# Or manage LocalStack manually for faster iteration:
make e2e-up          # Start LocalStack
make e2e-logs        # View LocalStack logs
cd tests/e2e && bash run-e2e-tests.sh  # Run tests manually
make e2e-down        # Stop LocalStack
make e2e-clean       # Stop and remove all artifacts
```

**Note:** The test script automatically sets AWS credentials (`AWS_ACCESS_KEY_ID=test`, `AWS_SECRET_ACCESS_KEY=test`, `AWS_DEFAULT_REGION=us-east-1`) for LocalStack if not already configured.

To manage LocalStack separately:

```bash
# Start LocalStack
make e2e-up

# Run tests manually
cd tests/e2e
bash run-e2e-tests.sh

# Stop LocalStack
cd ../..
make e2e-down

# Clean up test artifacts
make e2e-clean
```

### Option 2: Using Docker Compose Directly

```bash
# From the repository root
docker-compose up -d

# Wait for LocalStack to be ready
sleep 10

# Run the tests
cd tests/e2e
bash run-e2e-tests.sh

# Cleanup
cd ../..
docker-compose down
```

### Option 2: Full Manual Setup

```bash
# 1. Start LocalStack
docker run -d \
  --name fastssm-localstack \
  -p 4566:4566 \
  -e SERVICES=ssm,sts \
  -e DEBUG=1 \
  localstack/localstack:latest

# 2. Wait for LocalStack
sleep 15

# 3. Build and install the provider
go build -o terraform-provider-fastssm
PLUGIN_DIR="$HOME/.terraform.d/plugins/rumenvasilev/fastssm/dev/$(go env GOOS)_$(go env GOARCH)"
mkdir -p "$PLUGIN_DIR"
cp terraform-provider-fastssm "$PLUGIN_DIR/"

# 4. Run tests
cd tests/e2e
bash run-e2e-tests.sh

# 5. Cleanup
docker stop fastssm-localstack
docker rm fastssm-localstack
```

## Test Scenarios

### 1. Basic CRUD Operations (`main.tf`)

Tests creating various types of SSM parameters:

- String parameters
- StringList parameters
- SecureString parameters
- Parameters with descriptions
- Parameters with allowed patterns
- Parameters with insecure_value attribute
- Parameters with data_type specifications

Also validates data sources can read created parameters.

### 2. Import Functionality (`import.tf.template`)

Tests importing existing SSM parameters:

1. Creates a parameter directly via AWS CLI
2. Uses `terraform import` to bring it into state
3. Verifies the imported parameter matches expected values

### 3. Update Operations (`update.tf.template`)

Tests updating parameters:

1. Creates a parameter with initial values
2. Updates the parameter value and description
3. Verifies version increments correctly
4. Validates updated values are reflected

### 4. Destroy Operations

Tests that `terraform destroy` properly removes all created parameters.

## Test Structure

```
tests/e2e/
├── README.md                  # This file
├── run-e2e-tests.sh          # Main test orchestrator
├── main.tf                    # Primary test configuration (always loaded)
├── import.tf.template         # Import test template (copied when needed)
└── update.tf.template         # Update test template (copied when needed)
```

**Note**: `import.tf.template` and `update.tf.template` use the `.template` extension to prevent Terraform from loading them automatically. The test script copies them to `.tf` files when needed for specific tests.

## GitHub Actions Integration

The e2e tests run automatically on:

- Pull requests (except README-only changes)
- Pushes to main branch
- Manual workflow dispatch

The workflow uses the [official LocalStack GitHub Action](https://github.com/localstack/setup-localstack) which provides:
- Automatic LocalStack setup and startup
- Built-in `awslocal` CLI installation
- Proper health checks and wait mechanisms
- Optional state management with Cloud Pods

View workflow: `.github/workflows/e2e-test.yml`

## Troubleshooting

### LocalStack not starting

```bash
# Check LocalStack health
curl http://localhost:4566/_localstack/health

# Check LocalStack logs
docker logs fastssm-localstack

# Restart LocalStack
docker-compose restart localstack
```

### Provider not found

```bash
# Verify provider installation
ls -lh "$HOME/.terraform.d/plugins/rumenvasilev/fastssm/dev/$(go env GOOS)_$(go env GOARCH)/"

# Rebuild and reinstall
go build -o terraform-provider-fastssm
PLUGIN_DIR="$HOME/.terraform.d/plugins/rumenvasilev/fastssm/dev/$(go env GOOS)_$(go env GOARCH)"
mkdir -p "$PLUGIN_DIR"
cp terraform-provider-fastssm "$PLUGIN_DIR/"
```

### Terraform state issues

```bash
# Clean up state files
cd tests/e2e
rm -rf .terraform .terraform.lock.hcl terraform.tfstate*

# Re-initialize
terraform init
```

### AWS CLI connectivity

```bash
# Test AWS CLI can reach LocalStack
aws --endpoint-url=http://localhost:4566 ssm describe-parameters

# If it fails, check LocalStack is running
docker ps | grep localstack
```

## Environment Variables

The test script respects the following environment variables:

- `LOCALSTACK_URL`: LocalStack endpoint (default: `http://localhost:4566`)
- `PROVIDER_VERSION`: Provider version to use (default: `dev`)
- `TF_LOG`: Terraform log level (default: none, can set to `DEBUG`, `INFO`, etc.)

Example:

```bash
export TF_LOG=DEBUG
export PROVIDER_VERSION=1.0.0
bash run-e2e-tests.sh
```

## Adding New Tests

To add a new test scenario:

1. Create a new `.tf` file in `tests/e2e/` with your configuration
2. Add a new test section to `run-e2e-tests.sh`
3. Update this README with test description
4. Run tests locally to verify
5. Submit PR with changes

## Known Limitations

- LocalStack's SSM implementation may not support all AWS SSM features
- Some AWS-specific validations may not work identically
- Rate limiting behavior differs from real AWS

## Contributing

When contributing new tests, please ensure:

- Tests are idempotent and can run multiple times
- Cleanup is handled properly (via trap in script)
- Test names clearly describe what they validate
- Output is clear and informative

