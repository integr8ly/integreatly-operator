package monitoringcommon

const ObservatoriumFleetWideJSON = `{
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
	  "id": 5,
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
		  "id": 20,
		  "panels": [],
		  "title": "OSD & ROSA Cluster Count",
		  "type": "row"
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
			  "expr": "count(rhoam_cluster{}) by (type,externalID)",
			  "format": "time_series",
			  "instant": false,
			  "interval": "",
			  "legendFormat": "{{type, externalID}}",
			  "refId": "A"
			}
		  ],
		  "thresholds": [],
		  "timeFrom": null,
		  "timeRegions": [],
		  "timeShift": null,
		  "title": "OSD  and Rosa Clusters Count",
		  "tooltip": {
			"shared": true,
			"sort": 0,
			"value_type": "individual"
		  },
		  "transformations": [
			{
			  "id": "organize",
			  "options": {
				"excludeByName": {},
				"indexByName": {},
				"renameByName": {
				  "Time": "",
				  "{externalID=\"e99496aa-5b9b-40af-b64d-eab9bd55a793\", type=\"osd\"}": ""
				}
			  }
			}
		  ],
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
			"h": 3,
			"w": 6,
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
			"h": 3,
			"w": 6,
			"x": 18,
			"y": 4
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
		  "collapsed": false,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 7
		  },
		  "id": 22,
		  "panels": [],
		  "title": "RHOAM Version Count & Install Status",
		  "type": "row"
		},
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
			  },
			  {
				"matcher": {
				  "id": "byName",
				  "options": "Cluster ExternalID"
				},
				"properties": [
				  {
					"id": "custom.width",
					"value": 298
				  }
				]
			  }
			]
		  },
		  "gridPos": {
			"h": 8,
			"w": 24,
			"x": 0,
			"y": 8
		  },
		  "id": 9,
		  "interval": "100000",
		  "maxDataPoints": 1,
		  "options": {
			"showHeader": true,
			"sortBy": []
		  },
		  "pluginVersion": "7.3.10",
		  "targets": [
			{
			  "expr": "rhoam_cluster",
			  "format": "table",
			  "instant": true,
			  "interval": "",
			  "legendFormat": "",
			  "refId": "A"
			},
			{
			  "expr": "rhoam_version{}",
			  "format": "table",
			  "instant": true,
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
				  "externalID 2": true,
				  "instance": true,
				  "job 1": true,
				  "job 2": true,
				  "namespace 1": true,
				  "namespace 2": true,
				  "pod 1": true,
				  "pod 2": true,
				  "service 1": true,
				  "service 2": true,
				  "stage": true,
				  "to_version": false,
				  "version": false
				},
				"indexByName": {
				  "Time 1": 7,
				  "Time 2": 15,
				  "Value #A": 14,
				  "Value #B": 22,
				  "__name__ 1": 8,
				  "__name__ 2": 16,
				  "endpoint 1": 9,
				  "endpoint 2": 17,
				  "externalID 1": 5,
				  "externalID 2": 23,
				  "instance": 6,
				  "job 1": 10,
				  "job 2": 18,
				  "namespace 1": 11,
				  "namespace 2": 19,
				  "pod 1": 12,
				  "pod 2": 20,
				  "service 1": 13,
				  "service 2": 21,
				  "stage": 4,
				  "status": 1,
				  "to_version": 3,
				  "type": 0,
				  "version 1": 24,
				  "version 2": 2
				},
				"renameByName": {
				  "Time 1": "",
				  "Value #B": "",
				  "externalID": "Cluster External ID",
				  "externalID 1": "Cluster ExternalID",
				  "externalID 2": "",
				  "instance": "Rhoam instance",
				  "stage": "Stage",
				  "status": "Status",
				  "to_version": "Rhoam toVersion",
				  "type": "Cluster type",
				  "version": "Openshift  version",
				  "version 1": "Openshift version",
				  "version 2": "Rhoam version"
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
		},
		{
		  "collapsed": false,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 16
		  },
		  "id": 24,
		  "panels": [],
		  "title": "RHOAM AddOn Operator Versions count across clusters",
		  "type": "row"
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
			"y": 17
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
			  "expr": "count(rhoam_version{}) by (version, status) ",
			  "interval": "",
			  "legendFormat": "",
			  "refId": "A"
			}
		  ],
		  "thresholds": [],
		  "timeFrom": null,
		  "timeRegions": [],
		  "timeShift": null,
		  "title": "Rhoam AddOn Versions count ",
		  "tooltip": {
			"shared": false,
			"sort": 0,
			"value_type": "individual"
		  },
		  "transformations": [
			{
			  "id": "organize",
			  "options": {
				"excludeByName": {},
				"indexByName": {},
				"renameByName": {
				  "{stage=\"complete\", status=\"Installed\", version=\"1.20.0\"}": ""
				}
			  }
			}
		  ],
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
			"y": 17
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
			  "expr": "count(rhoam_version{}) by (version, status) ",
			  "instant": true,
			  "interval": "",
			  "legendFormat": "{{installed}}",
			  "refId": "A"
			}
		  ],
		  "title": "AddOn Version count ",
		  "type": "grafana-piechart-panel",
		  "valueName": "current"
		},
		{
		  "collapsed": false,
		  "datasource": null,
		  "gridPos": {
			"h": 1,
			"w": 24,
			"x": 0,
			"y": 23
		  },
		  "id": 5,
		  "panels": [],
		  "repeat": null,
		  "title": "RHOAM ALERTS across clusters",
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
			  },
			  {
				"matcher": {
				  "id": "byName",
				  "options": "Cluster External ID"
				},
				"properties": [
				  {
					"id": "custom.width",
					"value": 445
				  }
				]
			  },
			  {
				"matcher": {
				  "id": "byName",
				  "options": "Count"
				},
				"properties": [
				  {
					"id": "custom.width",
					"value": 68
				  }
				]
			  },
			  {
				"matcher": {
				  "id": "byName",
				  "options": "externalID"
				},
				"properties": [
				  {
					"id": "custom.width",
					"value": 264
				  }
				]
			  },
			  {
				"matcher": {
				  "id": "byName",
				  "options": "Cluster ExternalID"
				},
				"properties": [
				  {
					"id": "custom.width",
					"value": 309
				  }
				]
			  }
			]
		  },
		  "gridPos": {
			"h": 7,
			"w": 24,
			"x": 0,
			"y": 24
		  },
		  "id": 1,
		  "interval": "10s",
		  "links": [],
		  "options": {
			"frameIndex": 0,
			"showHeader": true,
			"sortBy": []
		  },
		  "pluginVersion": "7.3.10",
		  "targets": [
			{
			  "expr": "rhoam_alerts_summary{}",
			  "format": "table",
			  "instant": true,
			  "interval": "",
			  "intervalFactor": 1,
			  "legendFormat": "",
			  "refId": "A",
			  "step": 10
			},
			{
			  "expr": "rhoam_version{}",
			  "format": "table",
			  "instant": true,
			  "interval": "",
			  "legendFormat": "",
			  "refId": "C"
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
				  "Time 1": true,
				  "Time 2": true,
				  "Time 3": true,
				  "Value #B": true,
				  "Value #C": true,
				  "__name__": true,
				  "__name__ 1": true,
				  "__name__ 2": true,
				  "__name__ 3": true,
				  "endpoint": true,
				  "endpoint 1": true,
				  "endpoint 2": true,
				  "endpoint 3": true,
				  "instance": true,
				  "job": true,
				  "job 1": true,
				  "job 2": true,
				  "job 3": true,
				  "namespace": true,
				  "namespace 1": true,
				  "namespace 2": true,
				  "namespace 3": true,
				  "pod": true,
				  "pod 1": true,
				  "pod 2": true,
				  "pod 3": true,
				  "service": true,
				  "service 1": true,
				  "service 2": true,
				  "service 3": true
				},
				"indexByName": {
				  "Time 1": 4,
				  "Time 2": 11,
				  "Time 3": 27,
				  "Value #A": 26,
				  "Value #B": 22,
				  "Value #C": 23,
				  "__name__ 1": 5,
				  "__name__ 2": 12,
				  "__name__ 3": 28,
				  "alert": 0,
				  "endpoint 1": 6,
				  "endpoint 2": 13,
				  "endpoint 3": 29,
				  "externalID": 18,
				  "instance": 3,
				  "job 1": 7,
				  "job 2": 14,
				  "job 3": 30,
				  "namespace 1": 8,
				  "namespace 2": 15,
				  "namespace 3": 31,
				  "pod 1": 9,
				  "pod 2": 16,
				  "pod 3": 32,
				  "service 1": 10,
				  "service 2": 17,
				  "service 3": 33,
				  "severity": 1,
				  "stage": 21,
				  "state": 2,
				  "status": 20,
				  "type": 19,
				  "version 1": 25,
				  "version 2": 24
				},
				"renameByName": {
				  "Value": "Count",
				  "Value #A": "Count",
				  "Value #B": "",
				  "__name__": "",
				  "alert": "Alert",
				  "externalID": "Cluster External ID",
				  "service 1": "",
				  "severity": "Severity",
				  "stage": "Rhoam stage",
				  "state": "State",
				  "status": "Rhoam status",
				  "type": "Cluster type",
				  "version": "Rhoam version",
				  "version 1": "Openshift version",
				  "version 2": "Rhoam version"
				}
			  }
			},
			{
			  "id": "merge",
			  "options": {}
			},
			{
			  "id": "organize",
			  "options": {
				"excludeByName": {
				  "stage": true
				},
				"indexByName": {
				  "Value #A": 3,
				  "alert": 0,
				  "externalID": 4,
				  "severity": 1,
				  "stage": 9,
				  "state": 2,
				  "status": 8,
				  "to_version": 7,
				  "type": 5,
				  "version": 6
				},
				"renameByName": {
				  "Value #A": "Count",
				  "alert": "Alert",
				  "externalID": "Cluster ExternalID",
				  "severity": "Severity",
				  "stage": "Rhoam install stage",
				  "state": "State",
				  "status": "Rhoam status",
				  "to_version": "Rhoam toVersion",
				  "type": "Cluster type",
				  "version": "Rhoam version"
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
			"h": 8,
			"w": 19,
			"x": 0,
			"y": 31
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
			  "expr": "count(rhoam_alerts_summary{state=\"firing\", severity=\"critical\"}) by(severity, alert, externalID)",
			  "interval": "",
			  "legendFormat": "",
			  "refId": "A"
			}
		  ],
		  "thresholds": [],
		  "timeFrom": null,
		  "timeRegions": [],
		  "timeShift": null,
		  "title": "Count of Rhoam Critical alerts firing by severity and alertname",
		  "tooltip": {
			"shared": false,
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
			"h": 4,
			"w": 5,
			"x": 19,
			"y": 31
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
			  "expr": "count(rhoam_alerts_summary{state=\"firing\",severity=\"critical\"}) or vector(0)",
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
			"h": 4,
			"w": 5,
			"x": 19,
			"y": 35
		  },
		  "id": 52,
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
			  "expr": "count(rhoam_alerts_summary{state=\"pending\",severity=\"critical\"}) or vector(0)",
			  "instant": true,
			  "interval": "",
			  "legendFormat": "",
			  "refId": "A"
			}
		  ],
		  "title": "Total Criticals Pending",
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
			"y": 39
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
			  "expr": "count(rhoam_alerts_summary{state=\"firing\", severity=\"warning\"}) by(severity, alert, externalID)",
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
			"y": 39
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
			  "expr": "count(rhoam_alerts_summary{state=\"firing\",severity=\"warning\"}) or vector(0)",
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
	  "refresh": "10s",
	  "schemaVersion": 26,
	  "style": "dark",
	  "tags": [],
	  "templating": {
		"list": []
	  },
	  "time": {
		"from": "now-24h",
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
	  "title": "RHOAM Fleet Wide View",
	  "uid": "b06hC0U7z",
	  "version": 2
}`
