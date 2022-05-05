package monitoringcommon

const MonitoringGrafanaDBMultitenancyDetailedJSON = `{
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
  "id": 12,
  "links": [],
  "panels": [
    {
      "collapsed": false,
      "datasource": null,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 0
      },
      "id": 3,
      "panels": [],
      "title": "Tenants",
      "type": "row"
    },
    {
      "cacheTimeout": null,
      "datasource": "Prometheus",
      "description": "Number of active and reconciled APIManagementTenant CRs.",
      "fieldConfig": {
        "defaults": {
          "custom": {},
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 6,
        "w": 3,
        "x": 0,
        "y": 1
      },
      "id": 4,
      "interval": null,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "7.3.10",
      "targets": [
        {
          "expr": "num_reconciled_tenants",
          "instant": true,
          "interval": "",
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Active Tenant CRs",
      "type": "stat"
    },
    {
      "cacheTimeout": null,
      "datasource": "Prometheus",
      "description": "Number of APIManagementTenant CRs both reconciled and not reconciled.",
      "fieldConfig": {
        "defaults": {
          "custom": {},
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "blue",
                "value": null
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 6,
        "w": 3,
        "x": 3,
        "y": 1
      },
      "id": 5,
      "interval": null,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "7.3.10",
      "targets": [
        {
          "expr": "total_num_tenants",
          "instant": true,
          "interval": "",
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "Total Tenant CRs",
      "type": "stat"
    },
    {
      "datasource": null,
      "description": "Percentage of APIManagementTenant CRs that have been reconciled.",
      "fieldConfig": {
        "defaults": {
          "custom": {},
          "mappings": [],
          "thresholds": {
            "mode": "percentage",
            "steps": [
              {
                "color": "red",
                "value": null
              },
              {
                "color": "yellow",
                "value": 90
              },
              {
                "color": "green",
                "value": 95
              }
            ]
          },
          "unit": "percentunit"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 6,
        "w": 5,
        "x": 6,
        "y": 1
      },
      "id": 7,
      "options": {
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "showThresholdLabels": false,
        "showThresholdMarkers": true
      },
      "pluginVersion": "7.3.10",
      "targets": [
        {
          "expr": "num_reconciled_tenants / total_num_tenants",
          "instant": true,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "timeFrom": null,
      "timeShift": null,
      "title": "% Tenant CRs Reconciled",
      "type": "gauge"
    }
  ],
  "refresh": "10s",
  "schemaVersion": 26,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": []
  },
  "time": {
    "from": "now-1h",
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
    ],
    "time_options": [
      "5m",
      "15m",
      "1h",
      "6h",
      "12h",
      "24h",
      "2d",
      "7d",
      "30d"
    ]
  },
  "timezone": "",
  "title": "Multitenancy Detailed",
  "uid": "a5c77de61baa79315708d479f34ded2f8eb23381",
  "version": 1
}`
