
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
    print_step "Creating kind cluster..."
    if kind get clusters | grep -q "agent-cluster"; then
        print_warning "Deleting existing agent-cluster..."
        kind delete cluster --name agent-cluster
    fi
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

create_secret() {
    print_step "Creating Anthropic API secret..."
    kubectl create namespace agents --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic anthropic-secret \
        --from-literal=api-key="$ANTHROPIC_API_KEY" \
        --namespace agents \
        --dry-run=client -o yaml | kubectl apply -f -
    print_success "Secret created"
}

deploy_agents() {
    print_step "Deploying agents to Kubernetes..."
    kubectl apply -f k8s/doc-agent-deployment.yaml
    kubectl apply -f k8s/coder-agent-deployment.yaml
    kubectl apply -f k8s/nodeport-services.yaml
    print_step "Waiting for deployments to be ready..."
    kubectl rollout status deployment/doc-agent -n agents --timeout=300s
    kubectl rollout status deployment/coder-agent -n agents --timeout=300s
    print_success "All deployments ready"
}

show_status() {
    echo ""
    print_success "🎉 Multi-Agent System deployed successfully!"
    echo ""
    echo -e "${BLUE}📊 Cluster Status:${NC}"
    kubectl get nodes
    echo ""
    kubectl get pods -n agents -o wide
    echo ""
    kubectl get services -n agents
    echo ""
    echo -e "${BLUE}🔍 Testing Commands:${NC}"
    echo -e "  curl -X POST http://localhost:9080/coder -d 'search for encoding/json'${NC}"
    echo ""
    echo -e "  curl -X POST http://localhost:9081/doc -d 'encoding/json'${NC}"
    echo ""
    echo -e "  kubectl logs -f deployment/doc-agent -n agents${NC}"
    echo -e "  kubectl logs -f deployment/coder-agent -n agents${NC}"
    echo ""
    echo -e "  kubectl scale deployment doc-agent --replicas=6 -n agents${NC}"
    echo ""
    echo -e "  for i in {1..10}; do curl -X POST http://localhost:9080/coder -d 'test request'; done${NC}"
    echo ""
    echo -e "${BLUE}🧹 Cleanup:${NC}"
    echo -e "  kind delete cluster --name agent-cluster${NC}"
    echo ""
}

main() {
    echo -e "${BLUE}🚀 Setting up Multi-Agent System with Kind${NC}"
    echo ""
    create_secret
    deploy_agents
    show_status
}
main "$@"