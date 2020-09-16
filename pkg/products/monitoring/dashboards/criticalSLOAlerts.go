package monitoring

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

var MonitoringGrafanaDBCriticalSLOAlertsJSON = `{
	"annotations": {
		"list": [{
			"builtIn": 1,
			"datasource": "-- Grafana --",
			"enable": true,
			"hide": true,
			"iconColor": "rgba(0, 211, 255, 1)",
			"name": "Annotations & Alerts",
			"type": "dashboard"
		}]
	},
	"editable": true,
	"gnetId": null,
	"graphTooltip": 0,
	"id": 9,
	"iteration": 1586363497083,
	"links": [],
	"panels": [{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 0
			},
			"id": 2,
			"panels": [],
			"title": "SLO Summary (based on critical Alerts over the last 28 days & SLO of 99.9%)",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 1
			},
			"id": 4,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS {severity='critical', alertstate='firing'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 1
			},
			"id": 15,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertstate=\"firing\", severity=\"critical\"}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 1
			},
			"id": 12,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{severity='critical', alertstate='firing'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": 2,
			"description": "Amount of time left where at least 1 critical alert can be firing before the SLO is breached for the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 5
			},
			"id": 8,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "$slo_001_ms - (sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertstate=\"firing\", severity=\"critical\"})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000))",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0,0",
			"timeFrom": "28d",
			"hideTimeOverride": true, 
			"timeShift": null,
			"title": "Remaining Error Budget",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 5
			},
			"hideTimeOverride": true,
			"id": 100,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatedByRow": true,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertstate=\"firing\", severity=\"critical\"})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 9
			},
			"id": 48,
			"panels": [],
			"repeat": "product",
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 10
			},
			"id": 146,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 10
			},
			"id": 46,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 10
			},
			"id": 49,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 14
			},
			"hideTimeOverride": true,
			"id": 10,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 18
			},
			"id": 147,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 19
			},
			"id": 148,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 19
			},
			"id": 149,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 19
			},
			"id": 150,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 23
			},
			"hideTimeOverride": true,
			"id": 151,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 27
			},
			"id": 152,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 28
			},
			"id": 153,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 28
			},
			"id": 154,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 28
			},
			"id": 155,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 32
			},
			"hideTimeOverride": true,
			"id": 156,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 36
			},
			"id": 157,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 37
			},
			"id": 158,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 37
			},
			"id": 159,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 37
			},
			"id": 160,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 41
			},
			"hideTimeOverride": true,
			"id": 161,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 45
			},
			"id": 162,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 46
			},
			"id": 163,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 46
			},
			"id": 164,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 46
			},
			"id": 165,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 50
			},
			"hideTimeOverride": true,
			"id": 166,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 54
			},
			"id": 167,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 55
			},
			"id": 168,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 55
			},
			"id": 169,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 55
			},
			"id": 170,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 59
			},
			"hideTimeOverride": true,
			"id": 171,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 63
			},
			"id": 172,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 64
			},
			"id": 173,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 64
			},
			"id": 174,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 64
			},
			"id": 175,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 68
			},
			"hideTimeOverride": true,
			"id": 176,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 72
			},
			"id": 177,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 73
			},
			"id": 178,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 73
			},
			"id": 179,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 73
			},
			"id": 180,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 77
			},
			"hideTimeOverride": true,
			"id": 181,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 81
			},
			"id": 182,
			"panels": [],
			"repeat": null,
			"repeatIteration": 1586363497083,
			"repeatPanelId": 48,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush"
				}
			},
			"title": "$product",
			"type": "row"
		},
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
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 0,
				"y": 82
			},
			"id": 183,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 146,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "1,1",
			"timeFrom": null,
			"timeShift": null,
			"title": "Alerts Firing",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
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
			"decimals": 2,
			"description": "% of time where *no* critical alerts were firing over the last 28 days",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 82
			},
			"id": 184,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 46,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "clamp_max(\n    sum_over_time(\n        (clamp_max(\n            sum(absent(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}))\n            , 1\n        ))[28d:10m]\n    ) / (28 * 24 * 6) > 0, 1\n)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "0.999,0.999",
			"timeFrom": "28d",
			"hideTimeOverride": true,
			"timeShift": null,
			"title": "Overall SLO %",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"description": "Total number of critical alerts firing over the last 28 days. ",
			"fill": 1,
			"gridPos": {
				"h": 8,
				"w": 18,
				"x": 6,
				"y": 82
			},
			"id": 185,
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
			"links": [],
			"nullPointMode": "null",
			"options": {},
			"percentage": false,
			"pointradius": 2,
			"points": false,
			"renderer": "flot",
			"repeatIteration": 1586363497083,
			"repeatPanelId": 49,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush"
				}
			},
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'}) or vector(0)",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": [],
			"timeFrom": "28d",
			"timeRegions": [],
			"timeShift": null,
			"title": "Number of alerts firing ",
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
			"yaxes": [{
					"decimals": 0,
					"format": "none",
					"label": "",
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
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
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
			"decimals": null,
			"description": "Total time where at least 1 critical alert was firing over the last 28 days",
			"format": "ms",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 4,
				"w": 3,
				"x": 3,
				"y": 86
			},
			"hideTimeOverride": true,
			"id": 186,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
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
			"options": {},
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "0",
				"to": "null"
			}],
			"repeatIteration": 1586363497083,
			"repeatPanelId": 10,
			"repeatedByRow": true,
			"scopedVars": {
				"product": {
					"selected": false,
					"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush",
					"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush"
				}
			},
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "    sum_over_time(\n        (clamp_max(\n            sum(ALERTS{alertname=~\"[[product]].*\",alertstate = 'firing',severity = 'critical'} or ALERTS{namespace=~\"[[product]]donotmatch\",alertstate = 'firing',severity = 'critical'})\n            , 1\n        ))[28d:10m]\n    ) * (10 * 60 * 1000)",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"timeFrom": "28d",
			"timeShift": null,
			"title": "Firing Time ",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "0",
				"value": "null"
			}],
			"valueName": "current"
		}
	],
	"schemaVersion": 18,
	"style": "dark",
	"tags": [],
	"templating": {
		"list": [{
				"current": {
					"selected": true,
					"text": "28",
					"value": "28"
				},
				"hide": 2,
				"label": "SLO in days",
				"name": "slo_days",
				"options": [{
					"selected": true,
					"text": "28",
					"value": "28"
				}],
				"query": "28",
				"skipUrlSync": false,
				"type": "constant"
			},
			{
				"allValue": null,
				"current": {
					"selected": true,
					"text": "2419200000",
					"value": "2419200000"
				},
				"datasource": "Prometheus",
				"definition": "query_result(vector($slo_days * 24 * 60 * 60 * 1000))",
				"hide": 2,
				"includeAll": false,
				"label": "SLO in ms",
				"multi": false,
				"name": "slo_ms",
				"options": [{
					"selected": true,
					"text": "2419200000",
					"value": "2419200000"
				}],
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
					"selected": true,
					"text": "2416780800",
					"value": "2416780800"
				},
				"datasource": "Prometheus",
				"definition": "query_result(vector($slo_ms * 0.999))",
				"hide": 2,
				"includeAll": false,
				"label": "99.9% of SLO in ms",
				"multi": false,
				"name": "slo_999_ms",
				"options": [{
					"selected": true,
					"text": "2416780800",
					"value": "2416780800"
				}],
				"query": "query_result(vector($slo_ms * 0.999))",
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
					"selected": true,
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
				"options": [{
					"selected": true,
					"text": "2419200",
					"value": "2419200"
				}],
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
			},
			{
				"allValue": null,
				"current": {
					"text": "",
					"value": ""
				},
				"datasource": "Prometheus",
				"definition": "query_result(count(kube_namespace_labels{label_monitoring_key='middleware'}) by (namespace))",
				"hide": 2,
				"includeAll": false,
				"label": "namespace",
				"multi": false,
				"name": "namespace",
				"options": [{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `cloud-resources-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `cloud-resources-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `middleware-monitoring-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `middleware-monitoring-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups-operator"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-sso",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-sso"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-sso-operator",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-sso-operator"
					}
				],
				"query": "query_result(count(kube_namespace_labels{label_monitoring_key='middleware'}) by (namespace))",
				"refresh": 0,
				"regex": "/\"(.*?)\"/",
				"skipUrlSync": false,
				"sort": 1,
				"tagValuesQuery": "",
				"tags": [],
				"tagsQuery": "",
				"type": "query",
				"useTags": false
			},
			{
				"allValue": null,
				"current": {
					"selected": true,
					"text": "All",
					"value": "$__all"
				},
				"hide": 0,
				"includeAll": true,
				"label": "namespaceCustom",
				"multi": true,
				"name": "namespaceCustom",
				"options": [{
						"selected": true,
						"text": "All",
						"value": "$__all"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer"
					}
				],
				"query": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer",
				"skipUrlSync": false,
				"type": "custom"
			},
			{
				"allValue": null,
				"current": {
					"selected": true,
					"text": "All",
					"value": "$__all"
				},
				"hide": 0,
				"includeAll": true,
				"label": "product",
				"multi": true,
				"name": "product",
				"options": [{
						"selected": true,
						"text": "All",
						"value": "$__all"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito"
					},
					{
						"selected": false,
						"text": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush",
						"value": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush"
					}
				],
				"query": "` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `3scale|ThreeScale, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `amq-online|AMQ, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `fuse|Fuse, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `rhsso|Keycloak, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `user-rhsso|Keycloak, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `codeready-workspaces|CodeReady, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `solution-explorer|Solution, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `apicurito|Apicurito, ` + integreatlyv1alpha1.RHMISpec{}.NamespacePrefix + `ups|UnifiedPush",
				"skipUrlSync": false,
				"type": "custom"
			}
		]
	},
	"refresh": "10s",
	"time": {
		"from": "now-5m",
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
	"title": "Critical SLO summary",
	"uid": "eT5llOjWz",
	"version": 440
}`
