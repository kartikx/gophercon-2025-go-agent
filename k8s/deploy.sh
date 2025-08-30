#!/bin/bash

# Kubernetes Deployment Script for Multi-Agent System
set -e

echo "🚀 Deploying Multi-Agent System to Kubernetes..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl first."
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi

# Create namespace if it doesn't exist
kubectl create namespace agents --dry-run=client -o yaml | kubectl apply -f -

# Deploy secret (you need to update this with your actual API key)
echo "📝 Creating Anthropic API secret..."
echo "⚠️  Remember to update secret.yaml with your actual API key!"
kubectl apply -f k8s/secret.yaml -n agents

# Deploy documentation agent (4 replicas)
echo "📚 Deploying Documentation Agent (4 replicas)..."
kubectl apply -f k8s/doc-agent-deployment.yaml -n agents

# Deploy coder agent (2 replicas)  
echo "💻 Deploying Coder Agent (2 replicas)..."
kubectl apply -f k8s/coder-agent-deployment.yaml -n agents

# Wait for rollout
echo "⏳ Waiting for deployments to be ready..."
kubectl rollout status deployment/doc-agent -n agents
kubectl rollout status deployment/coder-agent -n agents

# Show status
echo "✅ Deployment complete!"
echo ""
echo "📊 Cluster Status:"
kubectl get pods -n agents -o wide
echo ""
kubectl get services -n agents

echo ""
echo "🔍 Testing commands:"
echo "  # Port forward to test coder agent:"
echo "  kubectl port-forward service/coder-agent-service 8080:8080 -n agents"
echo ""
echo "  # View logs:"
echo "  kubectl logs -f deployment/doc-agent -n agents"
echo "  kubectl logs -f deployment/coder-agent -n agents"
echo ""
echo "  # Scale agents:"
echo "  kubectl scale deployment doc-agent --replicas=6 -n agents"
echo ""
echo "🎉 Your multi-agent system is running with load balancing!"