logs: {
  level:  "info"
  pretty: false
}

forwards: [
  {
    name:      "test-forward-1"
    namespace: "default"
    resource:  "pod-1"
    ports: ["8080"]
  },
  {
    name:      "test-forward-2"
    namespace: "default"
    resource:  "pod-2"
    ports: ["9000"]
  },
]
