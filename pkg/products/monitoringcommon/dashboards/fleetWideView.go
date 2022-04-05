package monitoringcommon

const ObservatoriumFleetWideJSON =  `{
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
		  "collapsed": true,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 0
		  },
		  "id": 20,
		  "panels": [
			{
			  "aliasColors": {},
			  "bars": false,
			  "dashLength": 10,
			  "dashes": false,
			  "datasource": null,
			  "description": "",
			  "fieldConfig": {
				"defaults": {
				  "custom": {
					"align": null,
					"filterable": false
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
				"overrides": [
				  {
					"matcher": {
					  "id": "byName",
					  "options": "service 1"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 250
					  }
					]
				  }
				]
			  },
			  "fill": 1,
			  "fillGradient": 0,
			  "gridPos": {
				"h": 6,
				"w": 18,
				"x": 0,
				"y": 1
			  },
			  "hiddenSeries": false,
			  "id": 45,
			  "interval": "10s",
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
			  "maxDataPoints": 1150,
			  "nullPointMode": "null",
			  "options": {
				"alertThreshold": true
			  },
			  "percentage": false,
			  "pluginVersion": "7.3.10",
			  "pointradius": 2,
			  "points": false,
			  "renderer": "flot",
			  "seriesOverrides": [],
			  "spaceLength": 10,
			  "stack": false,
			  "steppedLine": false,
			  "targets": [
				{
				  "expr": "count(rhoam_cluster{type=\"osd\"})",
				  "format": "time_series",
				  "instant": false,
				  "interval": "",
				  "legendFormat": "Total # OSD Clusters",
				  "refId": "A"
				}
			  ],
			  "thresholds": [],
			  "timeFrom": null,
			  "timeRegions": [],
			  "timeShift": null,
			  "title": "OSD  Cluster Count",
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
			},
			{
			  "datasource": null,
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
			  "gridPos": {
				"h": 6,
				"w": 5,
				"x": 18,
				"y": 1
			  },
			  "id": 13,
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
				"text": {},
				"textMode": "auto"
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "expr": "count(rhoam_cluster{type=\"osd\"}) or vector(0)",
				  "instant": true,
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "timeFrom": null,
			  "timeShift": null,
			  "title": "OSD Cluster Count",
			  "type": "stat"
			},
			{
			  "aliasColors": {},
			  "bars": false,
			  "dashLength": 10,
			  "dashes": false,
			  "datasource": null,
			  "description": "",
			  "fieldConfig": {
				"defaults": {
				  "custom": {
					"align": null,
					"filterable": false
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
				"overrides": [
				  {
					"matcher": {
					  "id": "byName",
					  "options": "service 1"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 250
					  }
					]
				  }
				]
			  },
			  "fill": 1,
			  "fillGradient": 0,
			  "gridPos": {
				"h": 6,
				"w": 18,
				"x": 0,
				"y": 7
			  },
			  "hiddenSeries": false,
			  "id": 49,
			  "interval": "10s",
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
			  "maxDataPoints": 1150,
			  "nullPointMode": "null",
			  "options": {
				"alertThreshold": true
			  },
			  "percentage": false,
			  "pluginVersion": "7.3.10",
			  "pointradius": 2,
			  "points": false,
			  "renderer": "flot",
			  "seriesOverrides": [],
			  "spaceLength": 10,
			  "stack": false,
			  "steppedLine": false,
			  "targets": [
				{
				  "expr": "count(rhoam_cluster{type=\"rosa\"})",
				  "format": "time_series",
				  "instant": false,
				  "interval": "",
				  "legendFormat": "Total # OSD Clusters",
				  "refId": "A"
				}
			  ],
			  "thresholds": [],
			  "timeFrom": null,
			  "timeRegions": [],
			  "timeShift": null,
			  "title": "ROSA  Cluster Count",
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
			},
			{
			  "datasource": null,
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
			  "gridPos": {
				"h": 6,
				"w": 5,
				"x": 18,
				"y": 7
			  },
			  "id": 51,
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
				"text": {},
				"textMode": "auto"
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "expr": "count(rhoam_cluster{type=\"rosa\"}) or vector(0)",
				  "instant": true,
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "title": "ROSA Cluster Count",
			  "type": "stat"
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
				"h": 6,
				"w": 18,
				"x": 0,
				"y": 13
			  },
			  "hiddenSeries": false,
			  "id": 11,
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
			  "pluginVersion": "7.3.10",
			  "pointradius": 2,
			  "points": false,
			  "renderer": "flot",
			  "seriesOverrides": [],
			  "spaceLength": 10,
			  "stack": false,
			  "steppedLine": false,
			  "targets": [
				{
				  "exemplar": true,
				  "expr": "count(kube_node_labels) by (cluster_id)",
				  "interval": "",
				  "legendFormat": "{{cluster_id}}",
				  "refId": "A"
				}
			  ],
			  "thresholds": [],
			  "timeFrom": null,
			  "timeRegions": [],
			  "timeShift": null,
			  "title": "Cluster Node Count",
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
				  "$$hashKey": "object:2344",
				  "decimals": 0,
				  "format": "short",
				  "logBase": 1,
				  "min": "0",
				  "show": true
				},
				{
				  "$$hashKey": "object:2345",
				  "format": "short",
				  "logBase": 1,
				  "show": true
				}
			  ],
			  "yaxis": {
				"align": false
			  }
			},
			{
			  "datasource": null,
			  "description": "",
			  "fieldConfig": {
				"defaults": {
				  "custom": {
					"align": "auto",
					"displayMode": "auto",
					"filterable": false
				  },
				  "links": [
					{
					  "targetBlank": true,
					  "title": "OpenShift Console",
					  "url": "${__data.fields.url}"
					}
				  ],
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
				"overrides": [
				  {
					"matcher": {
					  "id": "byName",
					  "options": "url"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 561
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Value"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 72
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "#Kafka CRs"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 111
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "# Node Count"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 104
					  }
					]
				  }
				]
			  },
			  "gridPos": {
				"h": 6,
				"w": 5,
				"x": 18,
				"y": 13
			  },
			  "id": 15,
			  "links": [],
			  "maxDataPoints": 1,
			  "options": {
				"showHeader": true
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "exemplar": true,
				  "expr": "count(kube_node_labels * on(cluster_id) group_left(url) console_url) by(cluster_id, url)",
				  "instant": true,
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "title": "? OSD Cluster Node Count",
			  "transformations": [
				{
				  "id": "labelsToFields",
				  "options": {}
				},
				{
				  "id": "merge",
				  "options": {}
				},
				{
				  "id": "organize",
				  "options": {
					"excludeByName": {
					  "Time": true,
					  "cluster_id": true,
					  "url": false
					},
					"indexByName": {
					  "Time": 0,
					  "Value": 1,
					  "cluster_id": 2,
					  "url": 3
					},
					"renameByName": {
					  "Value": "# Node Count",
					  "cluster_id": "",
					  "url": "OpenShift Cluster (link)"
					}
				  }
				}
			  ],
			  "type": "table"
			}
		  ],
		  "title": "OSD & ROSA Cluster Count",
		  "type": "row"
		},
		{
		  "collapsed": true,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 1
		  },
		  "id": 22,
		  "panels": [
			{
			  "datasource": null,
			  "description": "",
			  "fieldConfig": {
				"defaults": {
				  "custom": {
					"align": null,
					"filterable": false
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
				"overrides": [
				  {
					"matcher": {
					  "id": "byName",
					  "options": "service 1"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 250
					  }
					]
				  }
				]
			  },
			  "gridPos": {
				"h": 8,
				"w": 24,
				"x": 0,
				"y": 2
			  },
			  "id": 9,
			  "interval": "100000",
			  "maxDataPoints": 1,
			  "options": {
				"showHeader": true,
				"sortBy": [
				  {
					"desc": false,
					"displayName": "Rhoam Status"
				  }
				]
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "expr": "rhoam_cluster",
				  "format": "table",
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				},
				{
				  "expr": "rhoam_version{}",
				  "format": "table",
				  "interval": "",
				  "legendFormat": "",
				  "refId": "B"
				}
			  ],
			  "timeFrom": null,
			  "timeShift": null,
			  "title": "Rhoam  Versions Count & Install Status",
			  "transformations": [
				{
				  "id": "seriesToColumns",
				  "options": {
					"byField": "instance"
				  }
				},
				{
				  "id": "organize",
				  "options": {
					"excludeByName": {
					  "Time 1": true,
					  "Time 2": true,
					  "Value #A": true,
					  "Value #B": true,
					  "__name__ 1": true,
					  "__name__ 2": true,
					  "endpoint 1": true,
					  "endpoint 2": true,
					  "job 1": true,
					  "job 2": true,
					  "namespace 2": true,
					  "pod 1": true,
					  "pod 2": true,
					  "service 1": false,
					  "service 2": true,
					  "version": false
					},
					"indexByName": {
					  "Time 1": 5,
					  "Time 2": 13,
					  "Value #A": 12,
					  "Value #B": 20,
					  "__name__ 1": 6,
					  "__name__ 2": 14,
					  "endpoint 1": 7,
					  "endpoint 2": 15,
					  "instance": 2,
					  "job 1": 8,
					  "job 2": 16,
					  "namespace 1": 9,
					  "namespace 2": 17,
					  "pod 1": 10,
					  "pod 2": 18,
					  "service 1": 11,
					  "service 2": 19,
					  "stage": 4,
					  "status": 3,
					  "type": 1,
					  "version": 0
					},
					"renameByName": {
					  "Time 1": "",
					  "instance": "",
					  "stage": "Rhoam Install Stage",
					  "status": "Rhoam Status",
					  "type": "Cluster type",
					  "version": "Rhoam version"
					}
				  }
				},
				{
				  "id": "seriesToColumns",
				  "options": {
					"byField": "version"
				  }
				}
			  ],
			  "type": "table"
			}
		  ],
		  "title": "RHOAM  Version Count & Install Status",
		  "type": "row"
		},
		{
		  "collapsed": true,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 2
		  },
		  "id": 24,
		  "panels": [
			{
			  "aliasColors": {},
			  "bars": false,
			  "dashLength": 10,
			  "dashes": false,
			  "datasource": null,
			  "fieldConfig": {
				"defaults": {
				  "custom": {},
				  "links": [
					{
					  "targetBlank": true,
					  "title": "Observability Routes in OpenShift Console",
					  "url": "${__field.labels.url}/k8s/ns/managed-application-services-observability/routes"
					}
				  ]
				},
				"overrides": []
			  },
			  "fill": 1,
			  "fillGradient": 0,
			  "gridPos": {
				"h": 6,
				"w": 19,
				"x": 0,
				"y": 3
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
			  "links": [],
			  "nullPointMode": "null",
			  "options": {
				"alertThreshold": true
			  },
			  "percentage": false,
			  "pluginVersion": "7.3.10",
			  "pointradius": 2,
			  "points": false,
			  "renderer": "flot",
			  "seriesOverrides": [],
			  "spaceLength": 10,
			  "stack": true,
			  "steppedLine": false,
			  "targets": [
				{
				  "exemplar": true,
				  "expr": "count(rhoam_version{}) by (version, status, stage) ",
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "thresholds": [],
			  "timeFrom": null,
			  "timeRegions": [],
			  "timeShift": null,
			  "title": "AddOn Versions count -  OSD clusters",
			  "tooltip": {
				"shared": false,
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
				  "format": "short",
				  "logBase": 1,
				  "min": "0",
				  "show": true
				},
				{
				  "format": "short",
				  "logBase": 1,
				  "show": true
				}
			  ],
			  "yaxis": {
				"align": false
			  }
			},
			{
			  "aliasColors": {},
			  "breakPoint": "50%",
			  "cacheTimeout": null,
			  "combine": {
				"label": "Others",
				"threshold": 0
			  },
			  "datasource": null,
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
			  "fontSize": "80%",
			  "format": "short",
			  "gridPos": {
				"h": 6,
				"w": 5,
				"x": 19,
				"y": 3
			  },
			  "id": 28,
			  "interval": null,
			  "legend": {
				"show": true,
				"values": true
			  },
			  "legendType": "Under graph",
			  "links": [],
			  "maxDataPoints": 1,
			  "nullPointMode": "connected",
			  "pieType": "pie",
			  "pluginVersion": "7.3.10",
			  "strokeWidth": 1,
			  "targets": [
				{
				  "expr": "count(rhoam_version{}) by (version, status, stage) ",
				  "instant": true,
				  "interval": "",
				  "legendFormat": "{{installed}}",
				  "refId": "A"
				}
			  ],
			  "title": "AddOn Version count - OSD Clusters",
			  "type": "grafana-piechart-panel",
			  "valueName": "current"
			},
			{
			  "aliasColors": {},
			  "bars": false,
			  "dashLength": 10,
			  "dashes": false,
			  "datasource": null,
			  "fieldConfig": {
				"defaults": {
				  "custom": {},
				  "links": [
					{
					  "targetBlank": true,
					  "title": "Observability Routes in OpenShift Console",
					  "url": "${__field.labels.url}/k8s/ns/managed-application-services-observability/routes"
					}
				  ]
				},
				"overrides": []
			  },
			  "fill": 1,
			  "fillGradient": 0,
			  "gridPos": {
				"h": 6,
				"w": 19,
				"x": 0,
				"y": 9
			  },
			  "hiddenSeries": false,
			  "id": 31,
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
			  "options": {
				"alertThreshold": true
			  },
			  "percentage": false,
			  "pluginVersion": "7.3.10",
			  "pointradius": 2,
			  "points": false,
			  "renderer": "flot",
			  "seriesOverrides": [],
			  "spaceLength": 10,
			  "stack": true,
			  "steppedLine": false,
			  "targets": [
				{
				  "exemplar": true,
				  "expr": "count(rhoam_version{}) by (version, status, stage) ",
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "thresholds": [],
			  "timeFrom": null,
			  "timeRegions": [],
			  "timeShift": null,
			  "title": "? AddOn Operator Versions count across ROSA clusters",
			  "tooltip": {
				"shared": false,
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
				  "format": "short",
				  "logBase": 1,
				  "min": "0",
				  "show": true
				},
				{
				  "format": "short",
				  "logBase": 1,
				  "show": true
				}
			  ],
			  "yaxis": {
				"align": false
			  }
			},
			{
			  "aliasColors": {},
			  "breakPoint": "50%",
			  "cacheTimeout": null,
			  "combine": {
				"label": "Others",
				"threshold": 0
			  },
			  "datasource": null,
			  "fieldConfig": {
				"defaults": {
				  "custom": {}
				},
				"overrides": []
			  },
			  "fontSize": "80%",
			  "format": "short",
			  "gridPos": {
				"h": 6,
				"w": 5,
				"x": 19,
				"y": 9
			  },
			  "id": 32,
			  "interval": null,
			  "legend": {
				"show": true,
				"values": true
			  },
			  "legendType": "Under graph",
			  "links": [],
			  "maxDataPoints": 1,
			  "nullPointMode": "connected",
			  "pieType": "pie",
			  "pluginVersion": "7.3.10",
			  "strokeWidth": 1,
			  "targets": [
				{
				  "expr": "count(rhoam_version{}) by (version, status, stage)",
				  "instant": true,
				  "interval": "",
				  "legendFormat": "{{installed}}",
				  "refId": "A"
				}
			  ],
			  "title": "AddOn Operator Versions count across ROSA clusters",
			  "type": "grafana-piechart-panel",
			  "valueName": "current"
			}
		  ],
		  "title": "RHOAM AddOn Operator Versions count across clusters",
		  "type": "row"
		},
		{
		  "collapsed": true,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 3
		  },
		  "id": 5,
		  "panels": [
			{
			  "datasource": "Prometheus",
			  "fieldConfig": {
				"defaults": {
				  "custom": {
					"align": null,
					"filterable": false
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
				"overrides": [
				  {
					"matcher": {
					  "id": "byName",
					  "options": "alert"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 228
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "namespace"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 198
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Value"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 66
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "state"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 110
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "severity"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 116
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "value"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 53
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Alert"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 233
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "State"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 89
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Severity"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 109
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "service"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 217
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "job"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 181
					  }
					]
				  }
				]
			  },
			  "gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 4
			  },
			  "id": 1,
			  "links": [],
			  "options": {
				"showHeader": true,
				"sortBy": [
				  {
					"desc": false,
					"displayName": "value"
				  }
				]
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "expr": "rhoam_alerts{}",
				  "format": "table",
				  "instant": true,
				  "interval": "",
				  "intervalFactor": 2,
				  "legendFormat": "",
				  "refId": "A",
				  "step": 10
				}
			  ],
			  "timeFrom": null,
			  "timeShift": null,
			  "title": "Rhoam Alerts across OSD and ROSA clusters",
			  "transformations": [
				{
				  "id": "organize",
				  "options": {
					"excludeByName": {
					  "Time": true,
					  "__name__": true,
					  "endpoint": false
					},
					"indexByName": {
					  "Time": 0,
					  "Value": 5,
					  "__name__": 1,
					  "alert": 2,
					  "endpoint": 11,
					  "instance": 8,
					  "job": 10,
					  "namespace": 6,
					  "pod": 7,
					  "service": 9,
					  "severity": 3,
					  "state": 4
					},
					"renameByName": {
					  "Value": "value",
					  "__name__": "",
					  "alert": "Alert",
					  "severity": "Severity",
					  "state": "State"
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
			  "datasource": null,
			  "fieldConfig": {
				"defaults": {
				  "custom": {},
				  "links": [
					{
					  "targetBlank": true,
					  "title": "Observability Routes in OpenShift Console",
					  "url": "${__field.labels.url}/k8s/ns/managed-application-services-observability/routes"
					}
				  ]
				},
				"overrides": []
			  },
			  "fill": 1,
			  "fillGradient": 0,
			  "gridPos": {
				"h": 5,
				"w": 19,
				"x": 0,
				"y": 11
			  },
			  "hiddenSeries": false,
			  "id": 34,
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
			  "options": {
				"alertThreshold": true
			  },
			  "percentage": false,
			  "pluginVersion": "7.3.10",
			  "pointradius": 2,
			  "points": false,
			  "renderer": "flot",
			  "seriesOverrides": [],
			  "spaceLength": 10,
			  "stack": true,
			  "steppedLine": false,
			  "targets": [
				{
				  "exemplar": true,
				  "expr": "count(ALERTS{alertstate=\"firing\", alertname!=\"DeadMansSwitch\", severity=\"critical\"}) by(severity, alertname) #count(ALERTS{alertstate=\"firing\", alertname!=\"DeadMansSwitch\", severity=\"critical\"} * on(cluster_id) group_left(url) console_url) by(severity, alertname, cluster_id, url)",
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "thresholds": [],
			  "timeFrom": null,
			  "timeRegions": [],
			  "timeShift": null,
			  "title": "Count of Critical alerts firing by severity and alertname",
			  "tooltip": {
				"shared": false,
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
				  "format": "short",
				  "logBase": 1,
				  "min": "0",
				  "show": true
				},
				{
				  "format": "short",
				  "logBase": 1,
				  "show": true
				}
			  ],
			  "yaxis": {
				"align": false
			  }
			},
			{
			  "datasource": null,
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
						"value": 1
					  }
					]
				  }
				},
				"overrides": []
			  },
			  "gridPos": {
				"h": 5,
				"w": 5,
				"x": 19,
				"y": 11
			  },
			  "id": 36,
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
				"text": {},
				"textMode": "auto"
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "expr": "count(ALERTS{alertstate=\"firing\",severity=\"critical\"}) or vector(0)",
				  "instant": true,
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "title": "Total Criticals Firing",
			  "type": "stat"
			},
			{
			  "aliasColors": {},
			  "bars": false,
			  "dashLength": 10,
			  "dashes": false,
			  "datasource": null,
			  "fieldConfig": {
				"defaults": {
				  "custom": {},
				  "links": [
					{
					  "targetBlank": true,
					  "title": "Observability Routes in OpenShift Console",
					  "url": "${__field.labels.url}/k8s/ns/managed-application-services-observability/routes"
					}
				  ]
				},
				"overrides": []
			  },
			  "fill": 1,
			  "fillGradient": 0,
			  "gridPos": {
				"h": 5,
				"w": 19,
				"x": 0,
				"y": 16
			  },
			  "hiddenSeries": false,
			  "id": 38,
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
			  "options": {
				"alertThreshold": true
			  },
			  "percentage": false,
			  "pluginVersion": "7.3.10",
			  "pointradius": 2,
			  "points": false,
			  "renderer": "flot",
			  "seriesOverrides": [],
			  "spaceLength": 10,
			  "stack": true,
			  "steppedLine": false,
			  "targets": [
				{
				  "exemplar": true,
				  "expr": "count(ALERTS{alertstate=\"firing\", alertname!=\"DeadMansSwitch\", severity=\"warning\"}) by(severity, alertname) #count(ALERTS{alertstate=\"firing\", alertname!=\"DeadMansSwitch\", severity=\"warning\"} * on(cluster_id) group_left(url) console_url) by(severity, alertname, cluster_id, url)",
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "thresholds": [],
			  "timeFrom": null,
			  "timeRegions": [],
			  "timeShift": null,
			  "title": "Count of Warning alerts firing by severity and alertname",
			  "tooltip": {
				"shared": false,
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
				  "format": "short",
				  "logBase": 1,
				  "min": "0",
				  "show": true
				},
				{
				  "format": "short",
				  "logBase": 1,
				  "show": true
				}
			  ],
			  "yaxis": {
				"align": false
			  }
			},
			{
			  "datasource": null,
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
						"value": 1
					  }
					]
				  }
				},
				"overrides": []
			  },
			  "gridPos": {
				"h": 5,
				"w": 5,
				"x": 19,
				"y": 16
			  },
			  "id": 40,
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
				"text": {},
				"textMode": "auto"
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "expr": "count(ALERTS{alertstate=\"firing\",severity=\"warning\"}) or vector(0)",
				  "instant": true,
				  "interval": "",
				  "legendFormat": "",
				  "refId": "A"
				}
			  ],
			  "title": "Total Warnings Firing",
			  "type": "stat"
			}
		  ],
		  "repeat": null,
		  "title": "RHOAM ALERTS across clusters",
		  "type": "row"
		},
		{
		  "collapsed": true,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 4
		  },
		  "id": 42,
		  "panels": [
			{
			  "datasource": "Prometheus",
			  "description": "Summary of CPU, Memory, Disk and Network usage for each Rhoam cluster.",
			  "fieldConfig": {
				"defaults": {
				  "custom": {
					"align": "left",
					"displayMode": "auto",
					"filterable": false
				  },
				  "links": [],
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
				"overrides": [
				  {
					"matcher": {
					  "id": "byName",
					  "options": "CPU Usage"
					},
					"properties": [
					  {
						"id": "unit"
					  },
					  {
						"id": "custom.width",
						"value": 118
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Memory Usage"
					},
					"properties": [
					  {
						"id": "unit",
						"value": "decbytes"
					  },
					  {
						"id": "custom.width",
						"value": 131
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Available Disk Space"
					},
					"properties": [
					  {
						"id": "unit",
						"value": "percent"
					  },
					  {
						"id": "custom.width",
						"value": 173
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Network Transmit Bytes"
					},
					"properties": [
					  {
						"id": "unit",
						"value": "decbytes"
					  },
					  {
						"id": "custom.width",
						"value": 193
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Network Received Bytes"
					},
					"properties": [
					  {
						"id": "unit",
						"value": "decbytes"
					  },
					  {
						"id": "custom.width",
						"value": 163
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "cluster_id"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 100
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "namespace"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 446
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "namespace"
					},
					"properties": [
					  {
						"id": "links",
						"value": [
						  {
							"title": "Drill Down",
							"url": "./d/c37212696a6c9b90acae3af21966521e06169e1c/strimzi-kafka-fleet-panel-compute-resources-namespace?&var-namespace=${__value.raw}"
						  }
						]
					  }
					]
				  },
				  {
					"matcher": {
					  "id": "byName",
					  "options": "Time 2"
					},
					"properties": [
					  {
						"id": "custom.width",
						"value": 311
					  }
					]
				  }
				]
			  },
			  "gridPos": {
				"h": 8,
				"w": 24,
				"x": 0,
				"y": 5
			  },
			  "id": 44,
			  "options": {
				"footer": {
				  "fields": "",
				  "reducer": [
					"sum"
				  ],
				  "show": false
				},
				"frameIndex": 0,
				"showHeader": true,
				"sortBy": []
			  },
			  "pluginVersion": "7.3.10",
			  "targets": [
				{
				  "expr": "sum(kube_pod_info{namespace !~\"openshift.*\", namespace !~\".*monitor.*\", namespace !~\".*operator.*\",namespace !~\".*obser.*\",namespace !~ \".*logging.*\",namespace !~ \".*minio.*\"}) by (namespace,cluster_id)",
				  "format": "table",
				  "instant": true,
				  "interval": "",
				  "intervalFactor": 2,
				  "legendFormat": "",
				  "refId": "A"
				},
				{
				  "expr": "sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{container!=\"\",namespace !~\"openshift.*\", namespace !~\".*monitor.*\", namespace !~\".*operator.*\",namespace !~\".*obser.*\",namespace !~ \".*logging.*\", namespace !~ \".*minio.*\"}) by (namespace)",
				  "format": "table",
				  "instant": true,
				  "interval": "",
				  "intervalFactor": 2,
				  "legendFormat": "",
				  "refId": "B"
				},
				{
				  "expr": "sum(container_memory_working_set_bytes{container != \"\", namespace !~\"openshift.*\", namespace !~\".*monitor.*\", namespace !~\".*operator.*\",namespace !~\".*obser.*\",namespace !~ \".*logging.*\",namespace !~ \".*minio.*\"}) by (namespace)",
				  "format": "table",
				  "instant": true,
				  "interval": "",
				  "intervalFactor": 2,
				  "legendFormat": "",
				  "refId": "C"
				},
				{
				  "expr": "sum((sum(kubelet_volume_stats_available_bytes{ job=\"kubelet\", namespace !~\"openshift.*\", namespace !~\".*monitor.*\", namespace !~\".*operator.*\",namespace !~\".*obser.*\",namespace !~ \".*logging.*\",namespace !~ \".*minio.*\"}) by (namespace) / sum(kubelet_volume_stats_capacity_bytes{ job=\"kubelet\", namespace !~\"openshift.*\", namespace !~\".*monitor.*\", namespace !~\".*operator.*\",namespace !~\".*obser.*\",namespace !~ \".*logging.*\",namespace !~ \".*minio.*\"}) by (namespace) * 100)) by (namespace) ",
				  "format": "table",
				  "instant": true,
				  "interval": "",
				  "intervalFactor": 2,
				  "legendFormat": "",
				  "refId": "D"
				},
				{
				  "expr": "sum(container_network_transmit_bytes_total{ container!= \"\",namespace !~\"openshift.*\", namespace !~\".*monitor.*\", namespace !~\".*operator.*\",namespace !~\".*obser.*\",namespace !~ \".*logging.*\",namespace !~ \".*minio.*\"}) by (namespace)",
				  "format": "table",
				  "instant": true,
				  "interval": "",
				  "intervalFactor": 2,
				  "legendFormat": "",
				  "refId": "E"
				},
				{
				  "expr": "sum(container_network_receive_bytes_total{ container!= \"\",namespace !~\"openshift.*\", namespace !~\".*monitor.*\", namespace !~\".*operator.*\",namespace !~\".*obser.*\",namespace !~ \".*logging.*\",namespace !~ \".*minio.*\"}) by (namespace)",
				  "format": "table",
				  "instant": true,
				  "interval": "",
				  "intervalFactor": 2,
				  "legendFormat": "",
				  "refId": "F"
				}
			  ],
			  "title": "Cluster Compute Metrics ",
			  "transformations": [
				{
				  "id": "seriesToColumns",
				  "options": {
					"byField": "namespace"
				  }
				},
				{
				  "id": "organize",
				  "options": {
					"excludeByName": {
					  "Time": true,
					  "Time 1": true,
					  "Time 2": true,
					  "Value #A": true,
					  "Value #C": false,
					  "cluster_id": true
					},
					"indexByName": {},
					"renameByName": {
					  "Value #B": "CPU Usage",
					  "Value #C": "Memory Usage",
					  "Value #D": "Available Disk Space",
					  "Value #E": "Network Transmit Bytes",
					  "Value #F": "Netowrk Received Bytes"
					}
				  }
				}
			  ],
			  "type": "table"
			}
		  ],
		  "title": "Compute Resources",
		  "type": "row"
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
		"from": "now-3h",
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
	  "title": "RHOAM - Fleet Wide View",
	  "uid": "b06hC0U7z",
	  "version": 40
}`