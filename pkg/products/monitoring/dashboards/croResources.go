package monitoring

const MonitoringGrafanaDBCROResourcesJSON = `{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": "-- Grafana --",
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "id": 23,
  "iteration": 1614258659586,
  "links": [],
  "panels": [
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "custom": {},
          "mappings": [
            {
              "from": "",
              "id": 0,
              "text": "Up",
              "to": "",
              "type": 1,
              "value": "1"
            },
            {
              "from": "",
              "id": 1,
              "text": "Down",
              "to": "",
              "type": 1,
              "value": "0"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "red",
                "value": null
              },
              {
                "color": "green",
                "value": 1
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 8,
        "x": 0,
        "y": 0
      },
      "id": 24,
      "options": {
        "colorMode": "value",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "value_and_name"
      },
      "pluginVersion": "7.2.0",
      "targets": [
        {
          "expr": "cro_postgres_available",
          "interval": "",
          "legendFormat": "{{productName}}",
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Postgres Connection",
      "type": "stat"
    },
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "custom": {},
          "mappings": [
            {
              "from": "",
              "id": 0,
              "text": "Up",
              "to": "",
              "type": 1,
              "value": "1"
            },
            {
              "from": "",
              "id": 1,
              "text": "Down",
              "to": "",
              "type": 1,
              "value": "0"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "red",
                "value": null
              },
              {
                "color": "green",
                "value": 1
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 8,
        "x": 8,
        "y": 0
      },
      "id": 23,
      "options": {
        "colorMode": "value",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "value_and_name"
      },
      "pluginVersion": "7.2.0",
      "targets": [
        {
          "expr": "cro_redis_available",
          "interval": "",
          "legendFormat": "{{resourceID}}",
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Redis Connection",
      "type": "stat"
    },
    {
      "collapsed": true,
      "datasource": "Prometheus",
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 5
      },
      "id": 5,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "Prometheus",
          "fieldConfig": {
            "defaults": {
              "custom": {
                "align": null
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  },
                  {
                    "color": "red",
                    "value": 80
                  }
                ]
              }
            },
            "overrides": []
          },
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 10,
            "w": 24,
            "x": 0,
            "y": 6
          },
          "hiddenSeries": false,
          "id": 2,
          "legend": {
            "alignAsTable": false,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "nullPointMode": "null",
          "percentage": false,
          "pluginVersion": "7.1.1",
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "cro_postgres_cpu_utilization_average",
              "format": "time_series",
              "hide": false,
              "interval": "",
              "legendFormat": "{{productName}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Postgres CPU Utilization",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "transformations": [],
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Postgres CPU",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "Prometheus",
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 6
      },
      "id": 12,
      "panels": [
        {
          "datasource": "Prometheus",
          "fieldConfig": {
            "defaults": {
              "custom": {
                "align": null
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  }
                ]
              },
              "unit": "bytes"
            },
            "overrides": [
              {
                "matcher": {
                  "id": "byName",
                  "options": "Value #B"
                },
                "properties": [
                  {
                    "id": "custom.width",
                    "value": 392
                  }
                ]
              },
              {
                "matcher": {
                  "id": "byName",
                  "options": "Allocated"
                },
                "properties": [
                  {
                    "id": "custom.width",
                    "value": 292
                  }
                ]
              },
              {
                "matcher": {
                  "id": "byName",
                  "options": "Allocated"
                },
                "properties": [
                  {
                    "id": "unit",
                    "value": "bytes"
                  }
                ]
              },
              {
                "matcher": {
                  "id": "byName",
                  "options": "Free"
                },
                "properties": [
                  {
                    "id": "unit",
                    "value": "bytes"
                  }
                ]
              }
            ]
          },
          "gridPos": {
            "h": 5,
            "w": 24,
            "x": 0,
            "y": 7
          },
          "id": 14,
          "options": {
            "showHeader": true,
            "sortBy": [
              {
                "desc": true,
                "displayName": "Allocated"
              }
            ]
          },
          "pluginVersion": "7.1.1",
          "targets": [
            {
              "expr": "cro_postgres_max_memory*1048576",
              "format": "table",
              "instant": true,
              "interval": "",
              "legendFormat": "",
              "refId": "B"
            },
            {
              "expr": "cro_postgres_freeable_memory_average",
              "format": "table",
              "instant": true,
              "interval": "",
              "legendFormat": "",
              "refId": "A"
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Postgres Memory",
          "transformations": [
            {
              "id": "seriesToColumns",
              "options": {
                "byField": "productName"
              }
            },
            {
              "id": "organize",
              "options": {
                "excludeByName": {
                  "Time": true,
                  "__name__": true,
                  "clusterID": true,
                  "endpoint": true,
                  "exported_namespace": true,
                  "instance": true,
                  "instanceID": true,
                  "job": true,
                  "namespace": true,
                  "pod": true,
                  "resourceID": true,
                  "service": true,
                  "strategy": true
                },
                "indexByName": {
                  "Time": 1,
                  "Value #A": 15,
                  "Value #B": 14,
                  "__name__": 2,
                  "clusterID": 3,
                  "endpoint": 4,
                  "exported_namespace": 5,
                  "instance": 6,
                  "instanceID": 7,
                  "job": 8,
                  "namespace": 9,
                  "pod": 10,
                  "productName": 0,
                  "resourceID": 11,
                  "service": 12,
                  "strategy": 13
                },
                "renameByName": {
                  "Value #A": "Free",
                  "Value #B": "Allocated",
                  "productName": "Product"
                }
              }
            },
            {
              "id": "calculateField",
              "options": {
                "alias": "Used",
                "binary": {
                  "left": "Allocated",
                  "operator": "-",
                  "reducer": "sum",
                  "right": "Free"
                },
                "mode": "binary",
                "reduce": {
                  "reducer": "diff"
                },
                "replaceFields": false
              }
            }
          ],
          "type": "table"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "Prometheus",
          "fieldConfig": {
            "defaults": {
              "custom": {},
              "unit": "bytes"
            },
            "overrides": []
          },
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 7,
            "w": 24,
            "x": 0,
            "y": 12
          },
          "hiddenSeries": false,
          "id": 10,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "nullPointMode": "null",
          "percentage": false,
          "pluginVersion": "7.1.1",
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "cro_postgres_freeable_memory_average",
              "instant": false,
              "interval": "",
              "legendFormat": "{{productName}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Postgres Free Memory",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Postgres Memory",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "Prometheus",
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 7
      },
      "id": 7,
      "panels": [
        {
          "datasource": "Prometheus",
          "fieldConfig": {
            "defaults": {
              "custom": {
                "align": null
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  }
                ]
              },
              "unit": "bytes"
            },
            "overrides": [
              {
                "matcher": {
                  "id": "byName",
                  "options": "Free"
                },
                "properties": [
                  {
                    "id": "custom.width",
                    "value": 483
                  }
                ]
              },
              {
                "matcher": {
                  "id": "byName",
                  "options": "Product"
                },
                "properties": [
                  {
                    "id": "custom.width",
                    "value": 532
                  }
                ]
              }
            ]
          },
          "gridPos": {
            "h": 5,
            "w": 24,
            "x": 0,
            "y": 8
          },
          "id": 25,
          "options": {
            "showHeader": true,
            "sortBy": []
          },
          "pluginVersion": "7.1.1",
          "targets": [
            {
              "expr": "cro_postgres_current_allocated_storage",
              "format": "table",
              "instant": true,
              "interval": "",
              "legendFormat": "",
              "refId": "A"
            },
            {
              "expr": "cro_postgres_free_storage_average",
              "format": "table",
              "instant": true,
              "interval": "",
              "legendFormat": "",
              "refId": "B"
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Postgres Storage",
          "transformations": [
            {
              "id": "seriesToColumns",
              "options": {
                "byField": "productName"
              }
            },
            {
              "id": "organize",
              "options": {
                "excludeByName": {
                  "Time": true,
                  "__name__": true,
                  "clusterID": true,
                  "endpoint": true,
                  "exported_namespace": true,
                  "instance": true,
                  "instanceID": true,
                  "job": true,
                  "namespace": true,
                  "pod": true,
                  "resourceID": true,
                  "service": true,
                  "strategy": true
                },
                "indexByName": {
                  "Time": 1,
                  "Value #A": 15,
                  "Value #B": 14,
                  "__name__": 2,
                  "clusterID": 3,
                  "endpoint": 4,
                  "exported_namespace": 5,
                  "instance": 6,
                  "instanceID": 7,
                  "job": 8,
                  "namespace": 9,
                  "pod": 10,
                  "productName": 0,
                  "resourceID": 11,
                  "service": 12,
                  "strategy": 13
                },
                "renameByName": {
                  "Value #A": "Allocated",
                  "Value #B": "Free",
                  "productName": "Product"
                }
              }
            },
            {
              "id": "calculateField",
              "options": {
                "mode": "reduceRow",
                "reduce": {
                  "reducer": "diff"
                }
              }
            },
            {
              "id": "organize",
              "options": {
                "excludeByName": {},
                "indexByName": {},
                "renameByName": {
                  "Difference": "Used"
                }
              }
            }
          ],
          "type": "table"
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "Prometheus",
          "fieldConfig": {
            "defaults": {
              "custom": {
                "align": null
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  },
                  {
                    "color": "red",
                    "value": 80
                  }
                ]
              },
              "unit": "bytes"
            },
            "overrides": []
          },
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 9,
            "w": 24,
            "x": 0,
            "y": 13
          },
          "hiddenSeries": false,
          "id": 3,
          "legend": {
            "alignAsTable": false,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "nullPointMode": "null",
          "percentage": false,
          "pluginVersion": "7.1.1",
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "cro_postgres_free_storage_average",
              "format": "time_series",
              "hide": false,
              "interval": "",
              "intervalFactor": 1,
              "legendFormat": "{{productName}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Postgres Free Storage",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "transformations": [],
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Postgres Storage",
      "type": "row"
    },
    {
      "collapsed": true,
      "datasource": "Prometheus",
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 8
      },
      "id": 19,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "Prometheus",
          "fieldConfig": {
            "defaults": {
              "custom": {}
            },
            "overrides": []
          },
          "fill": 1,
          "fillGradient": 0,
          "gridPos": {
            "h": 8,
            "w": 24,
            "x": 0,
            "y": 9
          },
          "hiddenSeries": false,
          "id": 17,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "nullPointMode": "null",
          "percentage": false,
          "pluginVersion": "7.1.1",
          "pointradius": 2,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "cro_redis_cpu_utilization_average",
              "interval": "",
              "legendFormat": "{{resourceID}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Redis CPU Utilization",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percent",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "title": "Redis CPU",
      "type": "row"
    },
    {
      "collapsed": false,
      "datasource": "Prometheus",
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 9
      },
      "id": 21,
      "panels": [],
      "title": "Redis Memory",
      "type": "row"
    },
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "custom": {
            "align": null,
            "filterable": false
          },
          "mappings": [
            {
              "from": "",
              "id": 0,
              "text": "Up",
              "to": "",
              "type": 1,
              "value": "1"
            },
            {
              "from": "",
              "id": 1,
              "text": "Down",
              "to": "",
              "type": 1,
              "value": "0"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "bytes"
        },
        "overrides": [
          {
            "matcher": {
              "id": "byName",
              "options": "Usage"
            },
            "properties": [
              {
                "id": "unit",
                "value": "percent"
              }
            ]
          },
          {
            "matcher": {
              "id": "byName",
              "options": "Free"
            },
            "properties": [
              {
                "id": "custom.width",
                "value": 402
              }
            ]
          }
        ]
      },
      "gridPos": {
        "h": 6,
        "w": 24,
        "x": 0,
        "y": 10
      },
      "id": 28,
      "options": {
        "showHeader": true,
        "sortBy": [
          {
            "desc": false,
            "displayName": "Usage"
          }
        ]
      },
      "pluginVersion": "7.2.0",
      "targets": [
        {
          "expr": "cro_redis_freeable_memory_average",
          "format": "table",
          "instant": true,
          "interval": "",
          "legendFormat": "",
          "refId": "A"
        },
        {
          "expr": "cro_redis_memory_usage_percentage_average",
          "format": "table",
          "instant": true,
          "interval": "",
          "legendFormat": "",
          "refId": "B"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Redis Memory",
      "transformations": [
        {
          "id": "seriesToColumns",
          "options": {
            "byField": "resourceID"
          }
        },
        {
          "id": "organize",
          "options": {
            "excludeByName": {
              "Time": true,
              "__name__": true,
              "clusterID": true,
              "endpoint": true,
              "exported_namespace": true,
              "instance": true,
              "instanceID": true,
              "job": true,
              "namespace": true,
              "pod": true,
              "productName": true,
              "service": true,
              "strategy": true
            },
            "indexByName": {},
            "renameByName": {
              "Value #A": "Free",
              "Value #B": "Usage"
            }
          }
        }
      ],
      "type": "table"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "custom": {}
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 8,
        "w": 24,
        "x": 0,
        "y": 16
      },
      "hiddenSeries": false,
      "id": 16,
      "legend": {
        "avg": false,
        "current": false,
        "max": false,
        "min": false,
        "show": true,
        "total": false,
        "values": false
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.2.0",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "expr": "cro_redis_memory_usage_percentage_average",
          "interval": "",
          "legendFormat": "{{resourceID}}",
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Redis Memory Usage",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "percent",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        },
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "custom": {},
          "unit": "bytes"
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 8,
        "w": 24,
        "x": 0,
        "y": 24
      },
      "hiddenSeries": false,
      "id": 26,
      "legend": {
        "avg": false,
        "current": false,
        "max": false,
        "min": false,
        "show": true,
        "total": false,
        "values": false
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.2.0",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "expr": "cro_redis_freeable_memory_average",
          "interval": "",
          "legendFormat": "{{resourceID}}",
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Redis Freeable Memory",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "bytes",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        },
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    }
  ],
  "refresh": false,
  "schemaVersion": 26,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "datasource": "Prometheus",
        "filters": [],
        "hide": 0,
        "label": "",
        "name": "Filters",
        "skipUrlSync": false,
        "type": "adhoc"
      }
    ]
  },
  "time": {
    "from": "now-6h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h",
      "2h",
      "1d"
    ]
  },
  "timezone": "",
  "title": "CRO Resources",
  "uid": "OMFxtSyGk",
  "version": 2
}`
