package monitoring

import (
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

func GetMonitoringGrafanaDBRhssoAvailabilityErrorBudgetBurnJSON(installationName string) string {
	quota := ``
	if installationName == resources.InstallationNames[string(v1alpha1.InstallationTypeManagedApi)] {
		quota = `, 
			{
				"datasource": "Prometheus",
				"enable": true,
				"expr": "count by (stage,quota,toQuota)(rhoam_quota{toQuota!=\"\"})",
				"hide": false,
				"iconColor": "#FADE2A",
				"limit": 100,
				"name": "Quota",
				"showIn": 0,
				"step": "",
				"tagKeys": "stage,quota,toQuota",
				"tags": "",
				"titleFormat": "Quota Change (million per day)",
				"type": "tags",
				"useValueForTime": false
			}`
	}
	return `{
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
			},
			{
				"datasource": "Prometheus",
				"enable": true,
				"expr": "count by (stage,version,to_version)(` + installationName + `_version{to_version!=\"\"})",
				"hide": false,
				"iconColor": "#FADE2A",
				"limit": 100,
				"name": "Upgrade",
				"showIn": 0,
				"step": "",
				"tagKeys": "stage,version,to_version",
				"tags": "",
				"titleFormat": "Upgrade",
				"type": "tags",
				"useValueForTime": false
			}` + quota + `
		  ]
		},
		"editable": true,
		"gnetId": null,
		"graphTooltip": 0,
		"id": 12,
		"iteration": 1619613992303,
		"links": [],
		"panels": [
		  {
			"cacheTimeout": null,
			"colorBackground": true,
			"colorValue": false,
			"colors": [
			  "#299c46",
			  "rgba(237, 129, 40, 0.89)",
			  "#C4162A"
			],
			"datasource": "Prometheus",
			"description": "Total number of critical alerts currently firing",
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"format": "none",
			"gauge": {
			  "maxValue": 100,
			  "minValue": 0,
			  "show": false,
			  "thresholdLabels": false,
			  "thresholdMarkers": true
			},
			"gridPos": {
			  "h": 2,
			  "w": 6,
			  "x": 0,
			  "y": 0
			},
			"id": 9,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [
			  {
				"name": "value to text",
				"value": 1
			  },
			  {
				"name": "range to text",
				"value": 2
			  }
			],
			"maxDataPoints": 100,
			"nullPointMode": "connected",
			"nullText": null,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [
			  {
				"from": "null",
				"text": "0",
				"to": "null"
			  }
			],
			"sparkline": {
			  "fillColor": "rgba(31, 118, 189, 0.18)",
			  "full": false,
			  "lineColor": "rgb(31, 120, 193)",
			  "show": false
			},
			"tableColumn": "",
			"targets": [
			  {
				"expr": "sum(ALERTS {alertname=~\"RH.*Rhsso.*ErrorBudgetBurn\", severity='warning', alertstate='firing', product=~'rhoam|rhmi', service=\"keycloak\"})",
				"format": "time_series",
				"instant": true,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "",
				"refId": "A"
			  }
			],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "5xx SSO Error Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [
			  {
				"op": "=",
				"text": "0",
				"value": "null"
			  }
			],
			"valueName": "current"
		  },
		  {
			"cacheTimeout": null,
			"colorBackground": true,
			"colorValue": false,
			"colors": [
			  "#C4162A",
			  "rgba(237, 129, 40, 0.89)",
			  "#299c46"
			],
			"datasource": null,
			"decimals": 2,
			"description": "% of time where *no*  5xx haproxy ErrorBudgetBurn alerts were firing over the last 28 days",
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"format": "percentunit",
			"gauge": {
			  "maxValue": 100,
			  "minValue": 0,
			  "show": false,
			  "thresholdLabels": false,
			  "thresholdMarkers": true
			},
			"gridPos": {
			  "h": 2,
			  "w": 6,
			  "x": 6,
			  "y": 0
			},
			"hideTimeOverride": true,
			"id": 11,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [
			  {
				"name": "value to text",
				"value": 1
			  },
			  {
				"name": "range to text",
				"value": 2
			  }
			],
			"maxDataPoints": 100,
			"nullPointMode": "connected",
			"nullText": null,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [
			  {
				"from": "null",
				"text": "0",
				"to": "null"
			  }
			],
			"sparkline": {
			  "fillColor": "rgba(31, 118, 189, 0.18)",
			  "full": false,
			  "lineColor": "rgb(31, 120, 193)",
			  "show": false
			},
			"tableColumn": "",
			"targets": [
			  {
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"RH.*sso.*ErrorBudgetBurn\", alertstate=\"firing\", product=~\"rhoam|rhmi\", service=\"keycloak\"}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "",
				"refId": "A"
			  }
			],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Overall  SLO % over last 28 days",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [
			  {
				"op": "=",
				"text": "0",
				"value": "null"
			  }
			],
			"valueName": "current"
		  },
		  {
			"cacheTimeout": null,
			"colorBackground": true,
			"colorValue": false,
			"colors": [
			  "#C4162A",
			  "rgba(237, 129, 40, 0.89)",
			  "#299c46"
			],
			"datasource": null,
			"decimals": 2,
			"description": "Amount of time left where at least 1 critical alert can be firing before the SLO is breached for the last 28 days",
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"format": "ms",
			"gauge": {
			  "maxValue": 100,
			  "minValue": 0,
			  "show": false,
			  "thresholdLabels": false,
			  "thresholdMarkers": true
			},
			"gridPos": {
			  "h": 2,
			  "w": 6,
			  "x": 12,
			  "y": 0
			},
			"hideTimeOverride": true,
			"id": 13,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [
			  {
				"name": "value to text",
				"value": 1
			  },
			  {
				"name": "range to text",
				"value": 2
			  }
			],
			"maxDataPoints": 100,
			"nullPointMode": "connected",
			"nullText": null,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [
			  {
				"from": "null",
				"text": "0",
				"to": "null"
			  }
			],
			"sparkline": {
			  "fillColor": "rgba(31, 118, 189, 0.18)",
			  "full": false,
			  "lineColor": "rgb(31, 120, 193)",
			  "show": false
			},
			"tableColumn": "",
			"targets": [
			  {
				"expr": "$slo_001_ms - (sum_over_time(\n        (clamp_max(\n     sum(ALERTS{alertname=~\"RH.*sso.*ErrorBudgetBurn\", service=\"keycloak\", alertstate=\"firing\", severity=\"warning\", product=~\"rhoam|rhmi\"})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000))",
				"format": "time_series",
				"instant": true,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "",
				"refId": "A"
			  }
			],
			"thresholds": "0,0",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Remaining Errors Budget",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [
			  {
				"op": "=",
				"text": "0",
				"value": "null"
			  }
			],
			"valueName": "current"
		  },
		  {
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
			  "#299c46",
			  "rgba(237, 129, 40, 0.89)",
			  "#d44a3a"
			],
			"datasource": null,
			"decimals": null,
			"description": "Total time where at least 1 5xx Error Alert was firing over the last 28 days",
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"format": "ms",
			"gauge": {
			  "maxValue": 100,
			  "minValue": 0,
			  "show": false,
			  "thresholdLabels": false,
			  "thresholdMarkers": true
			},
			"gridPos": {
			  "h": 2,
			  "w": 6,
			  "x": 18,
			  "y": 0
			},
			"hideTimeOverride": true,
			"id": 15,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [
			  {
				"name": "value to text",
				"value": 1
			  },
			  {
				"name": "range to text",
				"value": 2
			  }
			],
			"maxDataPoints": 100,
			"nullPointMode": "connected",
			"nullText": null,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [
			  {
				"from": "null",
				"text": "0",
				"to": "null"
			  }
			],
			"repeatedByRow": true,
			"sparkline": {
			  "fillColor": "rgba(31, 118, 189, 0.18)",
			  "full": false,
			  "lineColor": "rgb(31, 120, 193)",
			  "show": false
			},
			"tableColumn": "",
			"targets": [
			  {
				"expr": "    sum_over_time(\n        (clamp_max(\n     sum(ALERTS{alertname=~\"RH.*sso.*ErrorBudgetBurn\", service=\"keycloak\", alertstate=\"firing\", severity=\"warning\", product=~\"rhoam|rhmi\"})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "",
				"refId": "A"
			  }
			],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [
			  {
				"op": "=",
				"text": "0",
				"value": "null"
			  }
			],
			"valueName": "current"
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
			  "h": 7,
			  "w": 8,
			  "x": 0,
			  "y": 2
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
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\", code=\"5xx\"}[5m]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\"}[5m])))",
				"format": "time_series",
				"instant": false,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "rhsso",
				"refId": "A"
			  },
			  {
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\", code=\"5xx\"}[5m]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\"}[5m])))",
				"format": "time_series",
				"instant": false,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "user-sso",
				"refId": "B"
			  }
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "5min  - SSO Availability 5xx haproxy Errors rate ratio ",
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
				"min": "0",
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
			"datasource": null,
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"fill": 1,
			"fillGradient": 0,
			"gridPos": {
			  "h": 7,
			  "w": 8,
			  "x": 8,
			  "y": 2
			},
			"hiddenSeries": false,
			"id": 7,
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
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\", code=\"5xx\"}[30m]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\"}[30m])))",
				"format": "time_series",
				"instant": false,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "rhsso",
				"refId": "A"
			  },
			  {
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\", code=\"5xx\"}[30m]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\"}[30m])))",
				"format": "time_series",
				"instant": false,
				"interval": "",
				"intervalFactor": 1,
				"legendFormat": "user-sso",
				"refId": "B"
			  }
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "30 min - SSO Availability 5xx haproxy Errors rate ratio",
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
				"min": "0",
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
			"datasource": null,
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"fill": 1,
			"fillGradient": 0,
			"gridPos": {
			  "h": 7,
			  "w": 8,
			  "x": 16,
			  "y": 2
			},
			"hiddenSeries": false,
			"id": 5,
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
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\", code=\"5xx\"}[1h]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\"}[1h])))",
				"interval": "",
				"legendFormat": "rhsso",
				"refId": "A"
			  },
			  {
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\", code=\"5xx\"}[1h]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\"}[1h])))",
				"interval": "",
				"legendFormat": "user-sso",
				"refId": "B"
			  }
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "1 h - SSO Availability 5xx haproxy Errors rate ratio",
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
				"min": "0",
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
			"datasource": null,
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"fill": 1,
			"fillGradient": 0,
			"gridPos": {
			  "h": 7,
			  "w": 8,
			  "x": 0,
			  "y": 9
			},
			"hiddenSeries": false,
			"id": 4,
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
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\", code=\"5xx\"}[6h]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\"}[6h])))",
				"interval": "",
				"legendFormat": "rhsso",
				"refId": "A"
			  },
			  {
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\", code=\"5xx\"}[6h]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\"}[6h])))",
				"interval": "",
				"legendFormat": "user-sso",
				"refId": "B"
			  }
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "6 h - SSO Availability 5xx haproxy Errors rate ratio",
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
				"min": "0",
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
			"datasource": null,
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"fill": 1,
			"fillGradient": 0,
			"gridPos": {
			  "h": 7,
			  "w": 8,
			  "x": 8,
			  "y": 9
			},
			"hiddenSeries": false,
			"id": 6,
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
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\", code=\"5xx\"}[1d]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\"}[1d])))",
				"interval": "",
				"legendFormat": "rhsso",
				"refId": "A"
			  },
			  {
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\", code=\"5xx\"}[1d]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\"}[1d])))",
				"interval": "",
				"legendFormat": "user-sso",
				"refId": "B"
			  }
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "1 day - SSO Availability 5xx haproxy Errors rate ratio",
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
				"min": "0",
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
			"datasource": null,
			"fieldConfig": {
			  "defaults": {
				"custom": {}
			  },
			  "overrides": []
			},
			"fill": 1,
			"fillGradient": 0,
			"gridPos": {
			  "h": 7,
			  "w": 8,
			  "x": 16,
			  "y": 9
			},
			"hiddenSeries": false,
			"id": 3,
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
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\", code=\"5xx\"}[3d]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-rhsso\"}[3d])))",
				"interval": "",
				"legendFormat": "rhsso",
				"refId": "A"
			  },
			  {
				"expr": "sum( sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\", code=\"5xx\"}[3d]))\n    /sum(rate(haproxy_backend_http_responses_total{route=~\"^keycloak.*\", exported_namespace=~\"redhat-.*-user-sso\"}[3d])))",
				"interval": "",
				"legendFormat": "user-sso",
				"refId": "B"
			  }
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "3 days - SSO Availability 5xx haproxy Errors rate ratio",
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
				"min": "0",
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
		"refresh": "10s",
		"schemaVersion": 26,
		"style": "dark",
		"tags": [],
		"templating": {
		  "list": [
			{
			  "current": {
				"selected": false,
				"text": "28",
				"value": "28"
			  },
			  "hide": 2,
			  "label": "SLO in days",
			  "name": "slo_days",
			  "options": [
				{
				  "selected": true,
				  "text": "28",
				  "value": "28"
				}
			  ],
			  "query": "28",
			  "skipUrlSync": false,
			  "type": "constant"
			},
			{
			  "allValue": null,
			  "current": {
				"selected": false,
				"text": "2419200000",
				"value": "2419200000"
			  },
			  "datasource": "Prometheus",
			  "definition": "query_result(vector($slo_days * 24 * 60 * 60 * 1000))",
			  "hide": 2,
			  "includeAll": false,
			  "label": "slo in ms",
			  "multi": false,
			  "name": "slo_ms",
			  "options": [
				{
				  "selected": true,
				  "text": "2419200000",
				  "value": "2419200000"
				}
			  ],
			  "query": "query_result(vector($slo_days * 24 * 60 * 60 * 1000))",
			  "refresh": 0,
			  "regex": "/.*\\s(.*)\\s.*/",
			  "skipUrlSync": false,
			  "sort": 0,
			  "tagValuesQuery": "",
			  "tags": [],
			  "tagsQuery": "",
			  "type": "query",
			  "useTags": false
			},
			{
			  "allValue": null,
			  "current": {
				"selected": false,
				"text": "2419200",
				"value": "2419200"
			  },
			  "datasource": "Prometheus",
			  "definition": "query_result(vector($slo_ms * 0.001))",
			  "hide": 2,
			  "includeAll": false,
			  "label": "0.1% in ms",
			  "multi": false,
			  "name": "slo_001_ms",
			  "options": [
				{
				  "selected": true,
				  "text": "2419200",
				  "value": "2419200"
				}
			  ],
			  "query": "query_result(vector($slo_ms * 0.001))",
			  "refresh": 0,
			  "regex": "/.*\\s(.*)\\s.*/",
			  "skipUrlSync": false,
			  "sort": 0,
			  "tagValuesQuery": "",
			  "tags": [],
			  "tagsQuery": "",
			  "type": "query",
			  "useTags": false
			}
		  ]
		},
		"time": {
		  "from": "now-15m",
		  "to": "now"
		},
		"timepicker": {},
		"timezone": "",
		"title": "SLO SSO Availability for 5xx Errors",
		"uid": "AAqDgdrMk",
		"version": 11
	  }`
}
