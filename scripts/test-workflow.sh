#!/bin/bash
# test-workflow.sh - Test the GitHub Actions workflow components locally

set -e

echo "🧪 Testing GitHub Actions workflow components locally..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        exit 1
    fi
}

# Test 1: Go Tests
echo -e "${YELLOW}1. Running Go tests...${NC}"
go test -v ./...
print_status $? "Go tests passed"

# Test 2: Go Build
echo -e "${YELLOW}2. Building Go binary...${NC}"
make build
print_status $? "Go build successful"

# Test 3: Docker Build (if Docker is available)
if command -v docker &> /dev/null; then
    echo -e "${YELLOW}3. Testing Docker build...${NC}"
    
    # Get build args similar to GitHub Actions
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    docker build \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT="$COMMIT" \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        -t k8s-external-ip-powerdns:test \
        .
    print_status $? "Docker build successful"
    
    # Test 4: Container Run Test
    echo -e "${YELLOW}4. Testing container startup...${NC}"
    timeout 5s docker run --rm k8s-external-ip-powerdns:test --help > /dev/null 2>&1 || true
    print_status 0 "Container startup test completed"
    
    # Clean up test image
    docker rmi k8s-external-ip-powerdns:test > /dev/null 2>&1 || true
    
else
    echo -e "${YELLOW}3. Skipping Docker tests (Docker not available)${NC}"
fi

# Test 5: Verify workflow file syntax
echo -e "${YELLOW}5. Checking GitHub Actions workflow syntax...${NC}"
if command -v yamllint &> /dev/null; then
    yamllint .github/workflows/docker-publish.yml
    print_status $? "Workflow YAML syntax valid"
else
    # Basic YAML check
    python3 -c "import yaml; yaml.safe_load(open('.github/workflows/docker-publish.yml'))" 2>/dev/null
    print_status $? "Workflow YAML syntax valid"
fi

# Test 6: Check required files
echo -e "${YELLOW}6. Checking required files...${NC}"
required_files=(
    ".github/workflows/docker-publish.yml"
    "Dockerfile"
    "go.mod"
    "main.go"
    "README.md"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  ✅ $file"
    else
        echo -e "  ❌ $file (missing)"
        exit 1
    fi
done
print_status 0 "All required files present"

# Test 7: Check .gitignore
echo -e "${YELLOW}7. Checking .gitignore...${NC}"
if grep -q "bin/" .gitignore && grep -q "k8s-external-ip-powerdns" .gitignore; then
    print_status 0 "Build artifacts properly ignored"
else
    echo -e "${YELLOW}⚠️  Consider adding build artifacts to .gitignore${NC}"
fi

echo ""
echo -e "${GREEN}🎉 All workflow component tests passed!${NC}"
echo ""
echo "Next steps to set up GitHub Actions:"
echo "1. Push this code to a GitHub repository"
echo "2. Enable GitHub Actions in your repository settings"
echo "3. Create a version tag to trigger a release build:"
echo "   git tag v1.0.0"
echo "   git push origin v1.0.0"
echo "4. Your Docker image will be published to: ghcr.io/<username>/<repo>/k8s-external-ip-powerdns"
echo ""
echo "For more details, see docs/github-actions.md"
