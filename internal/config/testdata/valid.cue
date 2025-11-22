logs: {
  level: "info"
  pretty: true
}

forwards: [
  {
    name: "api"
    ports: ["8080:8080", "9000:9000"]
    namespace: "default"
    resource: "api-server"
  },
  {
    name: "database"
    ports: ["5432"]
    namespace: "default"
    resource: "postgres"
  }
]
