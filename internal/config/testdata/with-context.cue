context: "staging"

logs: {
  level: "info"
}

forwards: [
  {
    name: "api"
    ports: ["8080"]
    namespace: "default"
    resource: "api-server"
  }
]
