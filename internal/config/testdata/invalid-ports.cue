logs: {
  level: "info"
  pretty: true
}

forwards: [
  {
    name: "api"
    ports: ["99999:8080"]
    namespace: "default"
    resource: "api-server"
  }
]
