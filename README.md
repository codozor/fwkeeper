# fwkeeper

[![Go](https://github.com/codozor/fwkeeper/actions/workflows/go.yml/badge.svg)](https://github.com/codozor/fwkeeper/actions/workflows/go.yml)

Port forwarding made easy. A Kubernetes port forwarding tool that automatically maintains persistent port forwards to pods with automatic reconnection and failure recovery.

## Features

- 🔄 **Automatic Port Forwarding**: Maintains persistent port forwards to Kubernetes pods
- 🔁 **Automatic Reconnection**: Automatically reconnects on pod restarts or connection failures
- ⚙️ **Easy Configuration**: CUE-based configuration for simple pod forwarding setup
- 📊 **Structured Logging**: Comprehensive logging with configurable levels and pretty-printing
- 🔐 **Kubernetes Integration**: Seamless integration with your Kubernetes cluster (local or in-cluster)
- 🚀 **Multiple Forwards**: Support for multiple simultaneous port forwards

## Installation

### Prerequisites

- **Go 1.25.1 or later** - [Download Go](https://golang.org/dl/)
- **Access to a Kubernetes cluster** (local kubeconfig or in-cluster)

Verify Go is installed:

```bash
go version
```

### Install

Install fwkeeper directly:

```bash
go install github.com/codozor/fwkeeper@latest
```

This installs the binary to `$GOPATH/bin` (typically `~/go/bin`), which should already be in your PATH.

Verify installation:

```bash
fwkeeper --version
fwkeeper --help
```

If `fwkeeper` is not found, ensure `$GOPATH/bin` is in your PATH:

```bash
# Add to ~/.bashrc, ~/.zshrc, or ~/.bash_profile
export PATH="$HOME/go/bin:$PATH"
```

### Build from Source (Development)

For development or building locally:

```bash
git clone https://github.com/codozor/fwkeeper.git
cd fwkeeper
go build -o fwkeeper ./
./fwkeeper run -c fwkeeper.cue
```

## Quick Start

### 1. Create a Configuration File

Create a `fwkeeper.cue` file:

```cue
logs: {
  level: "debug"
  pretty: true
}

forwards: [
  {
    name: "postgres"
    ports: ["5432"]
    namespace: "default"
    resource: "postgres-pod"
  },
  {
    name: "api"
    ports: ["8080:8080", "9000:9000"]
    namespace: "api"
    resource: "api-server-deployment-abc123"
  }
]
```

### 2. Run fwkeeper

```bash
./fwkeeper run -c fwkeeper.cue
```

Or using the default `fwkeeper.cue`:

```bash
./fwkeeper run
```

### 3. Access Your Services

The port forwards are now active. Connect to your services:

```bash
# Connect to postgres
psql -h localhost -p 5432

# Connect to API
curl http://localhost:8080
```

## Configuration

### Configuration File Format (CUE)

fwkeeper uses [CUE](https://cuelang.org/) for configuration validation and schema enforcement.

#### Top-Level Structure

```cue
logs: { ... }
forwards: [ ... ]
```

#### Logs Configuration

```cue
logs: {
  level: "debug"      # "error", "warn", "info", "debug", "trace" (default: "info")
  pretty: true        # Pretty-print logs to console (default: false)
}
```

#### Port Forward Configuration

```cue
forwards: [
  {
    name: "service-name"              # Unique identifier for this forward
    ports: ["8080", "9000:3000"]      # Local:remote port mappings
    namespace: "default"              # Kubernetes namespace
    resource: "pod-name"              # Pod, Service, Deployment, StatefulSet, or DaemonSet
  },
  # ... more forwards
]
```

**Resource Reference Syntax:**
- `"pod-name"` - Direct pod reference
- `"svc/service-name"` or `"service/service-name"` - Kubernetes Service
- `"dep/deployment-name"` or `"deployment/deployment-name"` - Kubernetes Deployment
- `"sts/statefulset-name"` or `"statefulset/statefulset-name"` - Kubernetes StatefulSet
- `"ds/daemonset-name"` or `"daemonset/daemonset-name"` - Kubernetes DaemonSet

When using a Service, Deployment, StatefulSet, or DaemonSet, fwkeeper automatically finds and connects to the first running pod that matches the resource's selector.

**Port Mapping Syntax:**
- `"8080"` - Forward local port 8080 to pod port 8080
- `"8080:9000"` - Forward local port 8080 to pod port 9000

**Validation Rules:**
- Port numbers must be between 1 and 65535
- Each forward must have a unique name
- Namespace and resource must be specified
- Local ports must be unique across all forwards

### Environment Variables

```bash
# Override kubeconfig location
KUBECONFIG=/path/to/kubeconfig ./fwkeeper run

# Use default kubeconfig locations
# 1. $KUBECONFIG environment variable
# 2. ~/.kube/config
# 3. In-cluster config (if running inside Kubernetes)
```

## Usage

### Basic Commands

```bash
# Show help
./fwkeeper --help

# Run with default config (fwkeeper.cue)
./fwkeeper run

# Run with custom config file
./fwkeeper run -c /path/to/config.cue

# Show command help
./fwkeeper run --help
```

### Exit and Shutdown

Press `Ctrl+C` to gracefully stop fwkeeper. It will:
1. Cancel all active port forwards
2. Close connections
3. Print shutdown message

### Configuration Hot-Reload

fwkeeper automatically detects changes to your configuration file and reloads without interrupting active port forwards (when possible).

**Automatic Reload:**
- fwkeeper watches your config file for changes
- When the file is saved, the configuration is automatically reloaded
- New forwards are started, removed forwards are stopped, modified forwards are restarted

**Manual Reload:**
Send a SIGHUP signal to trigger manual reload:

```bash
# In another terminal, while fwkeeper is running
kill -HUP <pid>

# Or using pkill
pkill -HUP fwkeeper
```

**Config Reload Behavior:**
- **Invalid config** → Current configuration continues, error logged, reload skipped
- **New forwards** → Started automatically
- **Removed forwards** → Stopped gracefully
- **Modified forwards** → Restarted with new configuration
- **Unchanged forwards** → Continue running without interruption

**Example:**
1. Start fwkeeper: `./fwkeeper run -c fwkeeper.cue`
2. Edit `fwkeeper.cue` (add, remove, or modify forwards)
3. Save the file
4. fwkeeper detects the change and updates automatically
5. View the logs to see what changed

## Examples

### Example 1: Database Access

Forward to a PostgreSQL database pod:

```cue
logs: {
  level: "info"
  pretty: true
}

forwards: [
  {
    name: "database"
    ports: ["5432"]
    namespace: "databases"
    resource: "postgres-primary"
  }
]
```

Then connect:
```bash
psql -h localhost -p 5432 -U myuser -d mydb
```

### Example 2: Development Environment

Forward multiple services for local development:

```cue
logs: {
  level: "debug"
  pretty: true
}

forwards: [
  {
    name: "api-server"
    ports: ["8000:8000"]
    namespace: "development"
    resource: "api-server"
  },
  {
    name: "frontend"
    ports: ["3000:3000"]
    namespace: "development"
    resource: "frontend"
  },
  {
    name: "postgres"
    ports: ["5432:5432"]
    namespace: "databases"
    resource: "postgres-dev"
  },
  {
    name: "redis"
    ports: ["6379:6379"]
    namespace: "databases"
    resource: "redis-dev"
  }
]
```

### Example 3: Multiple Ports on Same Pod

Forward multiple ports from a single pod:

```cue
forwards: [
  {
    name: "api-services"
    ports: ["8000:8000", "9000:9000", "5000:5000"]
    namespace: "api"
    resource: "api-pod"
  }
]
```

### Example 4: Using Deployments and StatefulSets

Forward to pods managed by Deployments, StatefulSets, or DaemonSets:

```cue
logs: {
  level: "info"
  pretty: true
}

forwards: [
  {
    name: "api-deployment"
    ports: ["8080:8080"]
    namespace: "production"
    resource: "dep/api-server"        # Deployment: automatically finds a running pod
  },
  {
    name: "postgres-statefulset"
    ports: ["5432:5432"]
    namespace: "databases"
    resource: "sts/postgres-primary"  # StatefulSet: automatically finds a running pod
  },
  {
    name: "monitoring-daemonset"
    ports: ["9090:9090"]
    namespace: "monitoring"
    resource: "ds/prometheus"         # DaemonSet: automatically finds a running pod
  }
]
```

The resource finder automatically locates the first running pod managed by the specified Deployment, StatefulSet, or DaemonSet. This is useful when you have multiple replicas and want to connect to any available pod.

## Logging

### Log Levels

- **error**: Only errors
- **warn**: Warnings and errors
- **info**: General information (default)
- **debug**: Detailed debugging information
- **trace**: Very detailed tracing information

### Log Output

Logs are written to stderr. Each log entry includes:
- Timestamp (Unix milliseconds)
- Log level
- Message
- Error details (when applicable)

### Pretty Printing

Enable pretty-printed logs for development:

```cue
logs: {
  pretty: true
}
```

Output will include timestamps and color-coded levels for better readability.

## Troubleshooting

### "Pod not in running state"

The pod exists but isn't currently running. Check pod status:

```bash
kubectl get pods -n <namespace> <pod-name>
kubectl describe pod -n <namespace> <pod-name>
```

### "Connection refused" or "Connection reset"

The pod restarted or the port-forward connection dropped. fwkeeper will automatically reconnect with exponential backoff (starting at 100ms, up to 30s). Check logs for details.

### "Unable to connect to kubeconfig"

Verify your kubeconfig:

```bash
# Check KUBECONFIG env var
echo $KUBECONFIG

# Test cluster access
kubectl cluster-info

# Set correct kubeconfig
export KUBECONFIG=~/.kube/config
./fwkeeper run
```

### Configuration Validation Errors

Verify your CUE configuration syntax and against the schema:

```bash
# Check for CUE syntax errors in your config file
# Ensure all fields match the required structure in schema.cue
```

Common issues:
- Missing required fields (name, ports, namespace, resource)
- Invalid port numbers (not 1-65535)
- Malformed port specifications (should be "port" or "local:remote")

### Debug Logs

Enable debug logging to see detailed information:

```cue
logs: {
  level: "debug"
  pretty: true
}
```

## Development

### Project Structure

```
fwkeeper/
├── cmd/                      # CLI command definitions
│   ├── root.go              # Root command
│   └── run.go               # Run command
├── internal/
│   ├── app/                 # Application orchestration
│   │   └── runner.go        # Main runner and lifecycle management
│   ├── bootstrap/           # Dependency injection setup
│   ├── config/              # Configuration loading and validation
│   │   └── schema.cue       # CUE schema definition
│   ├── forwarder/           # Port forwarding logic
│   │   └── forwarder.go     # Individual pod port forwarder
│   ├── kubernetes/          # Kubernetes client setup
│   ├── locator/             # Pod discovery and location
│   │   └── locator.go       # Pod/service locator implementations
│   └── logger/              # Logging setup
├── main.go                  # Application entry point
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
├── fwkeeper.cue             # Default configuration
└── README.md                # This file
```

### Building

```bash
# Build the binary
go build -o fwkeeper ./

# Build with version info
go build -ldflags="-X main.version=v1.0.0" -o fwkeeper ./
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestName ./path/to/package
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run ./...
```

### Dependencies

Key dependencies:
- **[Cobra](https://github.com/spf13/cobra)**: CLI framework
- **[CUE](https://cuelang.org/)**: Configuration language
- **[Zerolog](https://github.com/rs/zerolog)**: Structured logging
- **[samber/do](https://github.com/samber/do)**: Dependency injection
- **[client-go](https://github.com/kubernetes/client-go)**: Kubernetes client library

## Architecture

### Key Components

1. **Runner**: Orchestrates port forwarders, manages context and graceful shutdown
2. **Forwarder**: Implements individual pod port forwarding with automatic reconnection
3. **Config**: CUE-based configuration parsing and validation
4. **Logger**: Structured logging with zerolog
5. **Kubernetes Integration**: Handles kubeconfig loading and client initialization

### Port Forward Flow

1. Read configuration from CUE file
2. Load Kubernetes credentials
3. For each forward:
   - Locate the pod
   - Verify pod is running
   - Establish SPDY connection to pod
   - Forward ports
   - Reconnect on failure (exponential backoff: 100ms → 30s with jitter)
4. Listen for interrupt signal (Ctrl+C)
5. Gracefully shutdown all forwarders

## License

This project is under MIT License

## Support

For issues, questions, or contributions, please visit the [repository](https://github.com/codozor/fwkeeper).

