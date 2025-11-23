logs: {
  level: "info"
  pretty: true
}

forwards: [
  {
    name: "api"
    ports: ["8080:8080"]
    namespace: "default"
    resource: "api-server"
  },
  {
    name: "api2"
    ports: ["8080:9000"]
    namespace: "default"
    resource: "api-server-2"
  }
]
