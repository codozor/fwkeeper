

#Port: string & =~"^([0-9]{1,5})(:[0-9]{1,5})?$"

#LogsLevel: "error" | "warn" | "info" | "debug" | "trace"

#LogsConfiguration: {
    level: *"info" |  #LogsLevel
    pretty: *false | bool
}

#PortForwardConfiguration: {
    name: string

    ports: [#Port, ...#Port]

    namespace: string
    resource: string
}

forwards: [...#PortForwardConfiguration]

logs: #LogsConfiguration
