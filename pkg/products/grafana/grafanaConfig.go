package grafana

func getCustomerMonitoringGrafanaRateLimitJSON(requestsPerUnit, activeQuota string) string {
	return `{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": {
          "type": "datasource",
          "uid": "grafana"
        },
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "target": {
          "limit": 100,
          "matchAny": false,
          "tags": [],
          "type": "dashboard"
        },
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "id": 1,
  "links": [],
  "liveNow": false,
  "panels": [
    {
      "collapsed": false,
      "datasource": {
        "type": "prometheus"
      },
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 0
      },
      "id": 20,
      "panels": [],
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "refId": "A"
        }
      ],
      "title": "RHOAM API Rate Limiting",
      "type": "row"
    },
    {
      "datasource": {
        "type": "prometheus"
      },
      "description": "",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "decimals": 0,
          "mappings": [
            {
              "options": {
                "match": "null",
                "result": {
                  "text": "N/A"
                }
              },
              "type": "special"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
               {
                "color": "#299c46",
                "value": null
              },
              {
                "color": "#299c46"
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 0,
        "y": 1
      },
      "id": 4,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "background",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "9.6.0",
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "sum(increase(authorized_calls[1m]) or vector(0)) + sum(increase(limited_calls[1m]) or vector(0))",
          "instant": true,
          "refId": "A"
        }
      ],
      "title": "Last 1 Minute - No. Requests",
      "transparent": true,
      "type": "stat"
    },
    {
      "datasource": {
        "type": "prometheus"
      },
      "description": "",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "decimals": 0,
          "mappings": [
            {
              "options": {
                "match": "null",
                "result": {
                  "text": "N/A"
                }
              },
              "type": "special"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "#299c46",
                "value": null
              },
              {
                "color": "rgba(237, 129, 40, 0.89)",
                "value": 1
              },
              {
                "color": "#d44a3a"
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 3,
        "y": 1
      },
      "id": 10,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "background",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "9.6.0",
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "sum(increase(limited_calls[1m])) > 0 or vector(0)",
          "instant": true,
          "refId": "A"
        }
      ],
      "title": "Last 1 Minute - Rejected",
      "type": "stat"
    },
    {
      "datasource": {
        "type": "prometheus"
      },
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "decimals": 0,
          "mappings": [
            {
              "options": {
                "match": "null",
                "result": {
                  "text": "N/A"
                }
              },
              "type": "special"
            }
          ],
          "max": 100,
          "min": 0,
          "thresholds": {
            "mode": "percentage",
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
          "unit": "percent"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 6,
        "y": 1
      },
      "id": 22,
      "interval": "",
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "none",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "9.6.0",
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "(sum(increase(limited_calls[1m])) > 0 or vector(0))/(sum(increase(authorized_calls[1m]) or vector(0)) + sum(increase(limited_calls[1m]) or vector(0)))*100 > 0 or vector(0)",
          "instant": true,
          "refId": "A"
        }
      ],
      "title": "Last 1 Minute - Rejected/Requests",
      "type": "stat"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": {
        "type": "prometheus"
      },
      "decimals": 0,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 4,
      "gridPos": {
        "h": 10,
        "w": 15,
        "x": 9,
        "y": 1
      },
      "hiddenSeries": false,
      "id": 2,
      "interval": "1m",
      "legend": {
        "alignAsTable": false,
        "avg": false,
        "current": false,
        "hideEmpty": false,
        "hideZero": false,
        "max": false,
        "min": false,
        "rightSide": false,
        "show": true,
        "total": false,
        "values": false
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null as zero",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "9.6.0",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "sum(increase(authorized_calls[1m]) or vector(0)) + sum(increase(limited_calls[1m]) or vector(0))",
          "instant": false,
          "interval": "30s",
          "legendFormat": "No. of Requests",
          "refId": "A"
        },
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "$perMinuteRequestsPerUnit",
          "instant": false,
          "interval": "30s",
          "legendFormat": "Active Quota - ` + activeQuota + ` Per Day - Rate Limit - ` + requestsPerUnit + ` per minute",
          "refId": "B"
        }
      ],
      "thresholds": [],
      "timeRegions": [],
      "title": "Per Minute API Requests",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "mode": "time",
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "short",
          "logBase": 1,
          "show": true
        },
        {
          "format": "short",
          "logBase": 1,
          "show": false
        }
      ],
      "yaxis": {
        "align": false
      }
    },
    {
      "datasource": {
        "type": "prometheus"
      },
      "description": "",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "decimals": 0,
          "mappings": [
            {
              "options": {
                "match": "null",
                "result": {
                  "text": "N/A"
                }
              },
              "type": "special"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "#299c46",
                "value": null
              },
              {
                "color": "#299c46"
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 0,
        "y": 6
      },
      "hideTimeOverride": false,
      "id": 6,
      "interval": "",
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "background",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "9.6.0",
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "sum(increase(authorized_calls[24h]) or vector(0)) + sum(increase(limited_calls[24h]) or vector(0)) > 0 or vector(0)",
          "instant": true,
          "refId": "A"
        }
      ],
      "title": "Last 24 Hours - No. Requests",
      "type": "stat"
    },
    {
      "datasource": {
        "type": "prometheus"
      },
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "decimals": 0,
          "mappings": [
            {
              "options": {
                "match": "null",
                "result": {
                  "text": "N/A"
                }
              },
              "type": "special"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "#299c46",
                "value": null
              },
              {
                "color": "rgba(237, 129, 40, 0.89)",
                "value": 1
              },
              {
                "color": "#d44a3a"
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 3,
        "y": 6
      },
      "hideTimeOverride": false,
      "id": 16,
      "interval": "",
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "background",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "first"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "9.6.0",
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "sum(increase(limited_calls[24h]) or vector(0))",
          "format": "time_series",
          "instant": true,
          "refId": "A"
        }
      ],
      "title": "Last 24 Hours - Rejected",
      "type": "stat"
    },
    {
      "datasource": {
        "type": "prometheus"
      },
      "description": "",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "decimals": 0,
          "mappings": [
            {
              "options": {
                "match": "null",
                "result": {
                  "text": "N/A"
                }
              },
              "type": "special"
            }
          ],
          "thresholds": {
            "mode": "percentage",
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
          "unit": "percent"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 6,
        "y": 6
      },
      "id": 18,
      "interval": "",
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "none",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "9.6.0",
      "targets": [
        {
          "datasource": {
            "type": "prometheus"
          },
          "expr": "(sum(increase(limited_calls[24h])) > 0 or vector(0))/(sum(increase(authorized_calls[24h]) or vector(0)) + sum(increase(limited_calls[24h]) or vector(0)))*100 > 0 or vector(0)",
          "instant": true,
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "title": "Last 24 Hours -  Rejected/Requests",
      "type": "stat"
    }
  ],
  "refresh": "1m",
  "schemaVersion": 36,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "hide": 2,
        "name": "perMinuteRequestsPerUnit",
        "query": "` + requestsPerUnit + `",
        "skipUrlSync": false,
        "type": "constant"
      }
    ]
  },
  "time": {
    "from": "now-12h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "5s",
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
  "title": "Rate Limiting",
  "uid": "66ab72e0d012aacf34f907be9d81cd9e",
  "version": 1,
  "weekStart": ""
}`
}

// The UID above is used to construct the url for the grafana dashboard in customer alerts. Please do not edit this value.
