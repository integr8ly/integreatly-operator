package resources

// The payload for a simple fuse integration based on timer and logger
const FuseIntegrationPayload = `
{
  "name": "test-integration",
  "tags": [
    "timer"
  ],
  "flows": [
    {
      "id": "-M30tHvcORdr7SjkRAjn",
      "name": "",
      "steps": [
        {
          "id": "-M30tMT1ORdr7SjkRAjn",
          "configuredProperties": {
            "period": "60000"
          },
          "metadata": {
            "configured": "true"
          },
          "action": {
            "id": "io.syndesis:timer-action",
            "name": "Simple",
            "description": "Specify an amount of time and its unit to periodically trigger integration execution. ",
            "descriptor": {
              "inputDataShape": {
                "kind": "none"
              },
              "outputDataShape": {
                "kind": "none"
              },
              "propertyDefinitionSteps": [
                {
                  "name": "Period",
                  "properties": {
                    "period": {
                      "componentProperty": false,
                      "defaultValue": "60000",
                      "deprecated": false,
                      "description": "Period",
                      "displayName": "Period",
                      "javaType": "long",
                      "kind": "parameter",
                      "labelHint": "Delay between each execution of the integration.",
                      "required": true,
                      "secret": false,
                      "type": "duration"
                    }
                  },
                  "description": "Period"
                }
              ],
              "configuredProperties": {
                "timerName": "syndesis-timer"
              },
              "componentScheme": "timer"
            },
            "actionType": "connector",
            "pattern": "From"
          },
          "connection": {
            "uses": 0,
            "id": "timer",
            "name": "Timer",
            "metadata": {
              "hide-from-connection-pages": "true"
            },
            "connector": {
              "id": "timer",
              "version": 4,
              "actions": [
                {
                  "id": "io.syndesis:timer-action",
                  "name": "Simple",
                  "description": "Specify an amount of time and its unit to periodically trigger integration execution. ",
                  "descriptor": {
                    "inputDataShape": {
                      "kind": "none"
                    },
                    "outputDataShape": {
                      "kind": "none"
                    },
                    "propertyDefinitionSteps": [
                      {
                        "name": "Period",
                        "properties": {
                          "period": {
                            "componentProperty": false,
                            "defaultValue": "60000",
                            "deprecated": false,
                            "description": "Period",
                            "displayName": "Period",
                            "javaType": "long",
                            "kind": "parameter",
                            "labelHint": "Delay between each execution of the integration.",
                            "required": true,
                            "secret": false,
                            "type": "duration"
                          }
                        },
                        "description": "Period"
                      }
                    ],
                    "configuredProperties": {
                      "timerName": "syndesis-timer"
                    },
                    "componentScheme": "timer"
                  },
                  "actionType": "connector",
                  "pattern": "From"
                },
                {
                  "id": "io.syndesis:timer-chron",
                  "name": "Cron",
                  "description": "Specify a cron utility expression for a more complex integration execution schedule.",
                  "descriptor": {
                    "inputDataShape": {
                      "kind": "none"
                    },
                    "outputDataShape": {
                      "kind": "none"
                    },
                    "propertyDefinitionSteps": [
                      {
                        "name": "cron",
                        "properties": {
                          "cron": {
                            "componentProperty": false,
                            "defaultValue": "0 0/1 * * * ?",
                            "deprecated": false,
                            "description": "A cron expression, for example the expression for every minute is 0 0/1 * * * ?",
                            "displayName": "Cron Expression",
                            "javaType": "string",
                            "kind": "parameter",
                            "labelHint": "Delay between scheduling (executing) the integration expressed as a cron expression",
                            "required": true,
                            "secret": false,
                            "type": "string"
                          }
                        },
                        "description": "Cron"
                      }
                    ],
                    "configuredProperties": {
                      "triggerName": "syndesis-quartz"
                    },
                    "componentScheme": "quartz2"
                  },
                  "actionType": "connector",
                  "pattern": "From"
                }
              ],
              "name": "Timer",
              "dependencies": [
                {
                  "type": "MAVEN",
                  "id": "io.syndesis.connector:connector-timer:1.8.6.fuse-750001-redhat-00002"
                },
                {
                  "type": "MAVEN",
                  "id": "org.apache.camel:camel-quartz2:2.21.0.fuse-750033-redhat-00001"
                }
              ],
              "metadata": {
                "hide-from-connection-pages": "true"
              },
              "description": "Trigger events based on an interval or a quartz expression",
              "icon": "assets:timer.svg"
            },
            "connectorId": "timer",
            "icon": "assets:timer.svg",
            "description": "Trigger integration execution based on an interval or a cron expression",
            "isDerived": false
          },
          "stepKind": "endpoint"
        },
        {
          "id": "-M30tNPcORdr7SjkRAjn",
          "configuredProperties": {
            "contextLoggingEnabled": "false",
            "bodyLoggingEnabled": "false"
          },
          "metadata": {
            "configured": "true"
          },
          "stepKind": "log",
          "name": "Log"
        }
      ],
      "tags": [
        "timer"
      ],
      "type": "PRIMARY"
    }
  ],
  "description": ""
}
`
