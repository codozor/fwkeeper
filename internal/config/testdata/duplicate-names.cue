logs: {
  level: "info"
  pretty: true
}

forwards: [
  {
    name: "api"
    ports: ["8080"]
    namespace: "default"
    resource: "api-server-1"
  },
  {
    name: "api"
    ports: ["9000"]
    namespace: "default"
    resource: "api-server-2"
  }
]
