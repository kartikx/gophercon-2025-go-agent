#!/bin/bash
set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_step() {
    echo -e "${BLUE}🔄 $1${NC}"
}
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}
print_warning() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_prereqs() {
    print_step "Checking prerequisites..."
    if ! command -v kind &> /dev/null; then
        print_error "kind is not installed. Install with: go install sigs.k8s.io/kind@latest"
        exit 1
    fi
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    if ! command -v docker &> /dev/null; then
        print_error "docker is not installed. Please install docker first."
        exit 1
    fi
    if [ -z "$ANTHROPIC_API_KEY" ]; then
        print_error "ANTHROPIC_API_KEY environment variable is not set"
        print_warning "Export your API key: export ANTHROPIC_API_KEY=your-key-here"
        exit 1
    fi
    print_success "All prerequisites met"
}

create_cluster() {
    print_step "Checking kind cluster..."
    if kind get clusters | grep -q "agent-cluster"; then
        print_warning "Cluster 'agent-cluster' already exists, reusing it..."
        if kubectl cluster-info --context kind-agent-cluster &> /dev/null; then
            print_success "Existing cluster is ready and accessible"
            return 0
        else
            print_warning "Existing cluster is not accessible, recreating..."
            kind delete cluster --name agent-cluster
        fi
    fi
    print_step "Creating new kind cluster..."
    kind create cluster --config kind-config.yaml --wait 300s
    kubectl cluster-info --context kind-agent-cluster
    print_success "Kind cluster created successfully"
}

build_and_load_images() {
    print_step "Building Docker images..."
    docker build -f Dockerfile.coder -t coder-agent:latest .
    docker build -f Dockerfile.doc -t doc-agent:latest .
    print_step "Loading images into kind cluster..."
    kind load docker-image coder-agent:latest --name agent-cluster
    kind load docker-image doc-agent:latest --name agent-cluster
    print_success "Images built and loaded"
}

main() {
    echo -e "${BLUE}🚀 Setting up Prerequisites for Multi-Agent System${NC}"
    echo ""
    check_prereqs
    create_cluster
    build_and_load_images
    echo ""
    print_success "🎉 Prerequisites setup completed!"
    echo ""
    echo -e "${BLUE}📋 Next Steps:${NC}"
    echo -e "  Run ${YELLOW}./deploy-k8s.sh${NC} to deploy the agents to Kubernetes"
    echo ""
    echo -e "${BLUE}🔍 Verify Setup:${NC}"
    echo -e "  ${YELLOW}kubectl get nodes${NC}"
    echo -e "  ${YELLOW}kubectl get pods -n agents${NC}"
    echo ""
}
main "$@"
