package grafana

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	gfSecurityAdminUser = "admin"
)

func ReconcileGrafanaSecrets(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	log := l.NewLogger()
	log.Info("reconciling Grafana configuration secrets")
	nsPrefix := installation.Spec.NamespacePrefix

	grafanaProxySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-k8s-proxy",
			Namespace: nsPrefix + "customer-monitoring",
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, client, grafanaProxySecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(grafanaProxySecret, installation)
		if grafanaProxySecret.Data == nil {
			grafanaProxySecret.Data = map[string][]byte{}
		}
		grafanaProxySecret.Data["session_secret"] = []byte(resources.GenerateRandomPassword(20, 2, 2, 2))
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	grafanaAdminCredsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-admin-credentials",
			Namespace: nsPrefix + "customer-monitoring",
		},
		Data: getAdminCredsSecretData(),
		Type: corev1.SecretTypeOpaque,
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, grafanaAdminCredsSecret, func() error {
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getAdminCredsSecretData() map[string][]byte {
	password := []byte(RandStringRunes(10))
	credentials := map[string][]byte{
		"GF_SECURITY_ADMIN_USER":     []byte(gfSecurityAdminUser),
		"GF_SECURITY_ADMIN_PASSWORD": password,
	}

	// Make the credentials available to the environment, similar is it was done in Grafana operator (resolve admin login issue?)
	err := os.Setenv("GF_SECURITY_ADMIN_USER", string(credentials["GF_SECURITY_ADMIN_USER"]))
	if err != nil {
		fmt.Printf("can't set credentials as environment vars")
		return credentials
	}
	err = os.Setenv("GF_SECURITY_ADMIN_PASSWORD", string(credentials["GF_SECURITY_ADMIN_PASSWORD"]))
	if err != nil {
		fmt.Printf("can't set credentials as environment vars (optional)")
		return credentials
	}

	return credentials
}

func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

func RandStringRunes(s int) string {
	b := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b)
}

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
                "color": "rgba(237, 129, 40, 0.89)",
                "value": null
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
      "pluginVersion": "9.0.9",
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
      "pluginVersion": "9.0.9",
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
      "pluginVersion": "9.0.9",
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
      "pluginVersion": "9.0.9",
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
                "color": "rgba(237, 129, 40, 0.89)",
                "value": null
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
      "pluginVersion": "9.0.9",
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
      "pluginVersion": "9.0.9",
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
      "pluginVersion": "9.0.9",
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
