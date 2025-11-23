logs: {
  level:  "info"
  pretty: false
}

forwards: [
  {
    name:      "test-forward"
    namespace: "default"
    resource:  "test-pod"
    ports: ["8080"]
  },
]
