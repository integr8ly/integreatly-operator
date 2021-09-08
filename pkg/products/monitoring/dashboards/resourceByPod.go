package monitoring

import (
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

// This dashboard json is dynamically configured based on installation type (rhmi or rhoam)
// The installation name taken from the v1alpha1.RHMI.ObjectMeta.Name
func GetMonitoringGrafanaDBResourceByPodJSON(namespacePrefix, installationName string) string {
	quota := ``
	if installationName == resources.InstallationNames[string(v1alpha1.InstallationTypeManagedApi)] || installationName == resources.InstallationNames[string(v1alpha1.InstallationTypeMultitenantManagedApi)] {
		quota = `,
			{
				"datasource": "Prometheus",
				"enable": true,
				"expr": "count by (quota,toQuota)(rhoam_quota{toQuota!=\"\"})",
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
		"list": [{
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
	"links": [],
	"panels": [{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 0
			},
			"id": 4,
			"panels": [],
			"repeat": null,
			"title": "CPU Usage",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 1,
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 1
			},
			"id": 0,
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
			"nullPointMode": "null as zero",
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
					"expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{namespace=~'$namespace', pod=~'$pod', container!='POD'}) by (pod)",
					"format": "time_series",
					"intervalFactor": 2,
					"legendFormat": "{{pod}}",
					"legendLink": null,
					"step": 10
				},
				{
					"expr": "kube_pod_container_resource_requests{namespace=~'$namespace', pod=~'$pod',resource='cpu'}",
					"format": "time_series",
					"intervalFactor": 2,
					"legendFormat": "{{pod}}",
					"legendLink": null,
					"step": 10
				},
				{
					"expr": "kube_pod_container_resource_limits{namespace=~'$namespace', pod=~'$pod', resource='cpu'}",
					"format": "time_series",
					"intervalFactor": 2,
					"legendFormat": "{{pod}} Limit",
					"legendLink": null,
					"step": 10
				}
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "CPU Usage",
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
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
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
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 8
			},
			"id": 5,
			"panels": [],
			"repeat": null,
			"title": "CPU Quota",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"columns": [],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 1,
			"fontSize": "100%",
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 9
			},
			"id": 1,
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
			"nullPointMode": "null as zero",
			"pageSize": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"scroll": true,
			"seriesOverrides": [],
			"showHeader": true,
			"sort": {
				"col": 0,
				"desc": true
			},
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"styles": [{
					"alias": "Time",
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"pattern": "Time",
					"type": "hidden"
				},
				{
					"alias": "CPU Usage",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #A",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "CPU Requests",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #B",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "CPU Requests %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #C",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "CPU Limits",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #D",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "CPU Limits %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #E",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "Container",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "container",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"pattern": "/.*/",
					"thresholds": [],
					"type": "string",
					"unit": "short"
				}
			],
			"targets": [{
					"expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{namespace=~'$namespace', pod=~'$pod', container!='POD'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "A",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_requests{namespace=~'$namespace', pod=~'$pod',resource='cpu'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "B",
					"step": 10
				},
				{
					"expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{namespace=~'$namespace', pod=~'$pod'}) by (container) / sum(kube_pod_container_resource_requests{namespace=~'$namespace', pod=~'$pod',resource='cpu'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "C",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_limits{namespace=~'$namespace', pod=~'$pod', resource='cpu'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "D",
					"step": 10
				},
				{
					"expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{namespace=~'$namespace', pod=~'$pod'}) by (container) / sum(kube_pod_container_resource_limits{namespace=~'$namespace', pod=~'$pod', resource='cpu'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "E",
					"step": 10
				}
			],
			"thresholds": [],
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Quota",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"transform": "table",
			"type": "table",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
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
			]
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 16
			},
			"id": 6,
			"panels": [],
			"repeat": null,
			"title": "Memory Usage",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 1,
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 17
			},
			"id": 2,
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
			"nullPointMode": "null as zero",
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"targets": [{
					"expr": "sum(container_memory_working_set_bytes{namespace=~'$namespace', pod=~'$pod', container !='',container !='POD'}) by (pod)",
					"format": "time_series",
					"intervalFactor": 2,
					"legendFormat": "{{pod}}",
					"legendLink": null,
					"step": 10
				},
				{
					"expr": "kube_node_status_allocatable{namespace=~'$namespace', pod=~'$pod', resource='memory'}",
					"format": "time_series",
					"intervalFactor": 2,
					"legendFormat": "{{pod}} Request",
					"legendLink": null,
					"step": 10
				},
				{
					"expr": "kube_pod_container_resource_limits{namespace=~'$namespace', pod=~'$pod',resource='memory'}",
					"format": "time_series",
					"intervalFactor": 2,
					"legendFormat": "{{pod}} Limit",
					"legendLink": null,
					"step": 10
				}
			],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "Memory Usage",
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
					"format": "bytes",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
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
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 24
			},
			"id": 7,
			"panels": [],
			"repeat": null,
			"title": "Memory Quota",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"columns": [],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 1,
			"fontSize": "100%",
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 25
			},
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
			"links": [],
			"nullPointMode": "null as zero",
			"pageSize": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"scroll": true,
			"seriesOverrides": [],
			"showHeader": true,
			"sort": {
				"col": 0,
				"desc": true
			},
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"styles": [{
					"alias": "Time",
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"pattern": "Time",
					"type": "hidden"
				},
				{
					"alias": "Memory Usage",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #A",
					"thresholds": [],
					"type": "number",
					"unit": "bytes"
				},
				{
					"alias": "Memory Requests",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #B",
					"thresholds": [],
					"type": "number",
					"unit": "bytes"
				},
				{
					"alias": "Memory Requests %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #C",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "Memory Limits",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #D",
					"thresholds": [],
					"type": "number",
					"unit": "bytes"
				},
				{
					"alias": "Memory Limits %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #E",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "Container",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "container",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"pattern": "/.*/",
					"thresholds": [],
					"type": "string",
					"unit": "short"
				}
			],
			"targets": [{
					"expr": "sum(container_memory_working_set_bytes{namespace=~'$namespace', pod=~'$pod', container !=''}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "A",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_requests{namespace=~'$namespace', pod=~'$pod',resource='memory'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "B",
					"step": 10
				},
				{
					"expr": "sum(container_memory_working_set_bytes{namespace=~'$namespace', pod=~'$pod'}) by (container) / sum(kube_pod_container_resource_requests{namespace=~'$namespace', pod=~'$pod',resource='memory'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "C",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_requests{namespace=~'$namespace', pod=~'$pod', resource='memory'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "D",
					"step": 10
				},
				{
					"expr": "sum(container_memory_working_set_bytes{namespace=~'$namespace', pod=~'$pod', container!=''}) by (container) / sum(kube_pod_container_resource_requests{namespace=~'$namespace', pod=~'$pod', resource='memory'}) by (container)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "E",
					"step": 10
				}
			],
			"thresholds": [],
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Quota",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"transform": "table",
			"type": "table",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
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
			]
		}
	],
	"refresh": "10s",
	"schemaVersion": 16,
	"style": "dark",
	"tags": [],
	"templating": {
		"list": [{
				"allValue": null,
				"datasource": "Prometheus",
				"definition": "",
				"hide": 0,
				"includeAll": false,
				"label": "namespace",
				"multi": false,
				"name": "namespace",
				"options": [],
				"query": "query_result(count(kube_namespace_labels{namespace=~'` + namespacePrefix + `.*'}) by (namespace))",
				"refresh": 1,
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
				"datasource": "Prometheus",
				"definition": "",
				"hide": 0,
				"includeAll": false,
				"label": "pod",
				"multi": false,
				"name": "pod",
				"options": [],
				"query": "label_values(kube_pod_info{namespace=~'$namespace'}, pod)",
				"refresh": 1,
				"regex": "",
				"skipUrlSync": false,
				"sort": 1,
				"tagValuesQuery": "",
				"tags": [],
				"tagsQuery": "",
				"type": "query",
				"useTags": false
			}
		]
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
	"title": "Resource Usage By Pod",
	"uid": "c84ae905b9f54268be6be82c9a5b7dd6",
	"version": 2
}`
}
