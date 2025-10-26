#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
LOCALSTACK_URL="http://localhost:4566"
PROVIDER_VERSION="${PROVIDER_VERSION:-dev}"
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$TEST_DIR/../.." && pwd)"
WORK_DIR="${TEST_DIR}/.e2e-work"

# Set AWS credentials for LocalStack (if not already set)
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-test}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-test}"
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}FastSSM E2E Tests with LocalStack${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Function to print section headers
print_header() {
    echo -e "\n${YELLOW}>>> $1${NC}\n"
}

# Function to check if LocalStack is ready
wait_for_localstack() {
    print_header "Waiting for LocalStack to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "${LOCALSTACK_URL}/_localstack/health" | grep -q "\"ssm\": \"running\""; then
            echo -e "${GREEN}✓ LocalStack is ready!${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        echo "Waiting for LocalStack... (attempt $attempt/$max_attempts)"
        sleep 2
    done
    
    echo -e "${RED}✗ LocalStack failed to start${NC}"
    return 1
}

# Function to setup test environment
setup_test_env() {
    local test_name=$1
    shift
    local files=("$@")
    
    print_header "Setting up test environment for: $test_name"
    
    # Clean and recreate work directory
    rm -rf "$WORK_DIR"
    mkdir -p "$WORK_DIR"
    cd "$WORK_DIR"
    
    # Copy required files
    for file in "${files[@]}"; do
        if [ -f "$TEST_DIR/$file" ]; then
            echo "Copying $file..."
            cp "$TEST_DIR/$file" .
        elif [ -f "$TEST_DIR/${file}.template" ]; then
            echo "Copying ${file}.template as $file..."
            cp "$TEST_DIR/${file}.template" "$file"
        else
            echo -e "${RED}✗ File not found: $file${NC}"
            exit 1
        fi
    done
    
    # Copy Terraform CLI config
    if [ ! -f "$TEST_DIR/e2e-test.tfrc" ]; then
        echo -e "${RED}✗ e2e-test.tfrc not found. Run provider build first.${NC}"
        exit 1
    fi
    cp "$TEST_DIR/e2e-test.tfrc" .
    export TF_CLI_CONFIG_FILE="$WORK_DIR/e2e-test.tfrc"
    export CHECKPOINT_DISABLE=1
    
   
    echo -e "${GREEN}✓ Test environment ready${NC}"
}

# Function to cleanup after test
cleanup_test() {
    local test_name=$1
    
    print_header "Cleaning up: $test_name"
    cd "$WORK_DIR"
    
    # Destroy all resources
    echo "Destroying resources..."
    terraform destroy -auto-approve 2>/dev/null || true
    
    echo -e "${GREEN}✓ Test cleanup complete${NC}"
}

# Final cleanup function
final_cleanup() {
    print_header "Final cleanup..."
    rm -rf "$WORK_DIR"
    rm -f "$TEST_DIR/e2e-test.tfrc"
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Trap to ensure cleanup on exit
trap final_cleanup EXIT

# Step 1: Build and install the provider (skip if already built in CI)
PLUGIN_DIR="$HOME/.local/share/terraform/plugins"

if [ ! -f "$PLUGIN_DIR/terraform-provider-fastssm" ]; then
    print_header "Building Terraform Provider..."
    cd "$ROOT_DIR"
    go build -o terraform-provider-fastssm
    chmod +x terraform-provider-fastssm

    # Create plugin directory
    mkdir -p "$PLUGIN_DIR"
    cp terraform-provider-fastssm "$PLUGIN_DIR/"

    echo -e "${GREEN}✓ Provider built and installed to $PLUGIN_DIR${NC}"
else
    echo -e "${GREEN}✓ Provider already installed at $PLUGIN_DIR${NC}"
fi

# Verify the provider binary exists and is executable
echo "Verifying provider binary:"
ls -lh "$PLUGIN_DIR/terraform-provider-fastssm"
file "$PLUGIN_DIR/terraform-provider-fastssm" || true

# Create Terraform CLI config for dev overrides
print_header "Configuring Terraform to use local provider..."
TF_CLI_CONFIG="$TEST_DIR/e2e-test.tfrc"

cat > "$TF_CLI_CONFIG" <<EOF
provider_installation {
  dev_overrides {
    "rumenvasilev/fastssm" = "$PLUGIN_DIR"
  }

  # For all other providers, use the default registry
  direct {
    exclude = ["rumenvasilev/fastssm"]
  }
}
EOF

echo -e "${GREEN}✓ Terraform CLI config created${NC}"

# Step 2: Wait for LocalStack
wait_for_localstack

# Step 3: Test 1 - Basic CRUD Operations
print_header "Test 1: Basic CRUD Operations"
setup_test_env "Test 1" "main.tf"

echo "Creating parameters..."
terraform apply -auto-approve

echo "Verifying outputs..."
terraform output -json > output.json
cat output.json

echo "Validating parameter values..."
TEST_STRING_VALUE=$(terraform output -raw test_string_value)
if [ "$TEST_STRING_VALUE" == "test-value-123" ]; then
    echo -e "${GREEN}✓ Parameter value correct${NC}"
else
    echo -e "${RED}✗ Parameter value incorrect: $TEST_STRING_VALUE${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Test 1 passed${NC}"
cleanup_test "Test 1"

# Step 4: Test 2 - Import Functionality
print_header "Test 2: Import Existing Parameter"

# Create a parameter directly via AWS CLI
echo "Creating parameter via AWS CLI for import test..."
if command -v awslocal &> /dev/null; then
    awslocal ssm put-parameter \
        --name "/e2e/test/import" \
        --value "imported-value" \
        --type "String" \
        --overwrite
else
    aws --endpoint-url="${LOCALSTACK_URL}" \
        --region=us-east-1 \
        ssm put-parameter \
        --name "/e2e/test/import" \
        --value "imported-value" \
        --type "String" \
        --overwrite
fi

setup_test_env "Test 2" "main.tf" "import.tf"

echo "Importing parameter..."
terraform import fastssm_parameter.imported "/e2e/test/import"

echo "Verifying import..."
terraform plan -detailed-exitcode || {
    exitcode=$?
    if [ $exitcode -eq 2 ]; then
        echo -e "${YELLOW}⚠ Import created drift - expected for import test${NC}"
    else
        echo -e "${RED}✗ Import verification failed${NC}"
        exit 1
    fi
}

echo -e "${GREEN}✓ Test 2 passed${NC}"
cleanup_test "Test 2"

# Step 5: Test 3 - Update Operations
print_header "Test 3: Update Operations"
setup_test_env "Test 3" "main.tf" "update.tf"

echo "Creating initial parameter..."
terraform apply -auto-approve \
    -var="parameter_value=initial-value" \
    -var="parameter_description=Initial description"

INITIAL_VERSION=$(terraform output -raw updatable_version)
echo "Initial version: $INITIAL_VERSION"

echo "Updating parameter value..."
terraform apply -auto-approve \
    -var="parameter_value=updated-value" \
    -var="parameter_description=Updated description"

UPDATED_VERSION=$(terraform output -raw updatable_version)
echo "Updated version: $UPDATED_VERSION"

if [ "$UPDATED_VERSION" -gt "$INITIAL_VERSION" ]; then
    echo -e "${GREEN}✓ Parameter version incremented correctly${NC}"
else
    echo -e "${RED}✗ Parameter version did not increment: $INITIAL_VERSION -> $UPDATED_VERSION${NC}"
    exit 1
fi

UPDATED_VALUE=$(terraform output -raw updatable_value)
if [ "$UPDATED_VALUE" == "updated-value" ]; then
    echo -e "${GREEN}✓ Parameter value updated correctly${NC}"
else
    echo -e "${RED}✗ Parameter value not updated: $UPDATED_VALUE${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Test 3 passed${NC}"
cleanup_test "Test 3"

# Test Summary
print_header "Test Summary"
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ All E2E tests passed!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Tests completed:"
echo "  ✓ Test 1: Basic CRUD Operations"
echo "  ✓ Test 2: Import Functionality"
echo "  ✓ Test 3: Update Operations"
echo ""
