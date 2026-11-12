// fwkeeper configuration example
//
// Copy this file to fwkeeper.cue (ignored by git) and adapt it:
//   cp fwkeeper.example.cue fwkeeper.cue
//   fwkeeper run

// Kubeconfig context used by all forwards (optional).
// Defaults to the current-context of the kubeconfig; the --context flag overrides it.
// Pinning it avoids opening tunnels to the wrong cluster after a `kubectl config use-context`.
context: "dev-cluster"

logs: {
	level:  "info" // "error" | "warn" | "info" | "debug" | "trace"
	pretty: true   // human-readable console output
}

forwards: [
	// Service: forwards to a ready pod backing the service.
	// Remote ports are service ports, mapped to the pod target ports (numeric or named).
	{
		name:      "postgres"
		ports:     ["5432"] // local 5432 -> service port 5432
		namespace: "database"
		resource:  "svc/postgres"
	},

	// Deployment: forwards to a ready pod of the deployment, several ports at once.
	{
		name:      "api"
		ports:     ["8080:80", "9090"] // local 8080 -> pod 80, local 9090 -> pod 9090
		namespace: "backend"
		resource:  "deployment/api"
	},

	// StatefulSet (also available: ds/<name> for a DaemonSet).
	{
		name:      "redis"
		ports:     ["6379"]
		namespace: "cache"
		resource:  "sts/redis"
	},

	// Pod by name: no prefix.
	{
		name:      "debug-pod"
		ports:     ["5005"]
		namespace: "backend"
		resource:  "api-debug"
	},
]
