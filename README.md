# React-Go Agents

A Go-based agent system with support for documentation and coding agents.

## Local Development

```bash
# Build and run locally
go build .
./react-go  # defaults to doc agent on port 8080

# Or set environment variables
AGENT_TYPE=coder PORT=8081 ./react-go
```

## Docker Usage

### Build Images

```bash
# Build documentation agent
docker build -f Dockerfile.doc -t doc-agent .

# Build coder agent  
docker build -f Dockerfile.coder -t coder-agent .
```

### Run with Docker Compose

```bash
# Start both agents
docker-compose up

# Test the agents
curl -X POST http://localhost:8082/coder -d "read data directory"
curl -X POST http://localhost:8083/doc -d "search for fmt package"
```

### Run Individual Containers

```bash
# Documentation agent
docker run -p 8082:8080 -e AGENT_TYPE=doc -e PORT=8080 doc-agent

# Coder agent
docker run -p 8083:8080 -e AGENT_TYPE=coder -e PORT=8080 coder-agent
```

## Kubernetes Deployment

```bash
# Apply deployments
kubectl apply -f k8s/doc-agent-deployment.yaml
kubectl apply -f k8s/coder-agent-deployment.yaml

# Check status
kubectl get pods
kubectl get services

# Port forward for testing
kubectl port-forward service/doc-agent-service 8082:8080
kubectl port-forward service/coder-agent-service 8083:8080
```

## Environment Variables

- `AGENT_TYPE`: Type of agent (`doc` or `coder`)
- `PORT`: Port to listen on (default: 8080)

## Agent Communication

Agents can communicate with each other using their service names in Kubernetes:
- Documentation agent: `http://doc-agent-service:8080`
- Coder agent: `http://coder-agent-service:8080`

## API Endpoints

Both agents expose a POST endpoint at their root path:
- Documentation agent: `POST /doc`
- Coder agent: `POST /coder`

Send requests with plain text body containing your query.
