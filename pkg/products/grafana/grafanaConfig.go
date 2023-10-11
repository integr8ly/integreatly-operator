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
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "customer-monitoring"
	defaultRoutename             = "grafana-route"
	gfSecurityAdminUser          = "admin"
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
		fmt.Printf("can't set credentials as environment vars (optional)")
		return credentials
	}
	err = os.Setenv("GF_SECURITY_ADMIN_PASSWORD", string(credentials["GF_SECURITY_ADMIN_PASSWORD"]))
	if err != nil {
		fmt.Printf("can't set credentials as environment vars (optional)")
		return credentials
	}

	return credentials
}

func GetGrafanaConsoleURL(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (string, error) {

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	grafanaRoute := &routev1.Route{}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultRoutename, Namespace: ns}, grafanaRoute)
	if err != nil {
		return "", err
	}

	return "https://" + grafanaRoute.Spec.Host, nil
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
  "id": 2,
  "iteration": 1605286887536,
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
      "title": "RHOAM API Rate Limiting",
      "type": "row"
    },
    {
      "cacheTimeout": null,
      "colorBackground": true,
      "colorPostfix": false,
      "colorPrefix": false,
      "colorValue": false,
      "colors": [
        "#299c46",
        "rgba(237, 129, 40, 0.89)",
        "#d44a3a"
      ],
      "datasource": null,
      "description": "",
      "format": "none",
      "gauge": {
        "maxValue": 100,
        "minValue": 0,
        "show": false,
        "thresholdLabels": false,
        "thresholdMarkers": true
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 0,
        "y": 1
      },
      "id": 4,
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
      "options": {},
      "postfix": "",
      "postfixFontSize": "50%",
      "prefix": "",
      "prefixFontSize": "50%",
      "rangeMaps": [
        {
          "from": "null",
          "text": "N/A",
          "to": "null"
        }
      ],
      "sparkline": {
        "fillColor": "rgba(31, 118, 189, 0.18)",
        "full": false,
        "lineColor": "rgb(31, 120, 193)",
        "show": false,
        "ymax": null,
        "ymin": null
      },
      "tableColumn": "",
      "targets": [
        {
          "expr": "sum(increase(authorized_calls[1m]) or vector(0)) + sum(increase(limited_calls[1m]) or vector(0))",
          "instant": true,
          "refId": "A"
        }
      ],
      "thresholds": "$perMinuteRequestsPerUnit",
      "timeFrom": null,
      "timeShift": null,
      "title": "Last 1 Minute - No. Requests",
      "transparent": true,
      "type": "singlestat",
      "valueFontSize": "80%",
      "valueMaps": [
        {
          "op": "=",
          "text": "N/A",
          "value": "null"
        }
      ],
      "valueName": "avg"
    },
    {
      "cacheTimeout": null,
      "colorBackground": true,
      "colorValue": false,
      "colors": [
        "#299c46",
        "rgba(237, 129, 40, 0.89)",
        "#d44a3a"
      ],
      "datasource": null,
      "description": "",
      "format": "none",
      "gauge": {
        "maxValue": 100,
        "minValue": 0,
        "show": false,
        "thresholdLabels": false,
        "thresholdMarkers": true
      },
      "gridPos": {
        "h": 5,
        "w": 3,
        "x": 3,
        "y": 1
      },
      "id": 10,
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
      "options": {},
      "postfix": "",
      "postfixFontSize": "50%",
      "prefix": "",
      "prefixFontSize": "50%",
      "rangeMaps": [
        {
          "from": "null",
          "text": "N/A",
          "to": "null"
        }
      ],
      "sparkline": {
        "fillColor": "rgba(31, 118, 189, 0.18)",
        "full": false,
        "lineColor": "rgb(31, 120, 193)",
        "show": false,
        "ymax": null,
        "ymin": null
      },
      "tableColumn": "",
      "targets": [
        {
          "expr": "sum(increase(limited_calls[1m])) > 0 or vector(0)",
          "instant": true,
          "refId": "A"
        }
      ],
      "thresholds": "1",
      "timeFrom": null,
      "timeShift": null,
      "title": "Last 1 Minute - Rejected",
      "type": "singlestat",
      "valueFontSize": "80%",
      "valueMaps": [
        {
          "op": "=",
          "text": "N/A",
          "value": "null"
        }
      ],
      "valueName": "avg"
    },
    {
      "cacheTimeout": null,
      "colorBackground": false,
      "colorPostfix": false,
      "colorValue": false,
      "colors": [
        "#299c46",
        "rgba(237, 129, 40, 0.89)",
        "#d44a3a"
      ],
      "datasource": null,
      "format": "none",
      "gauge": {
        "maxValue": 100,
        "minValue": 0,
        "show": false,
        "thresholdLabels": false,
        "thresholdMarkers": true
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
      "options": {},
      "postfix": "%",
      "postfixFontSize": "50%",
      "prefix": "",
      "prefixFontSize": "50%",
      "rangeMaps": [
        {
          "from": "null",
          "text": "N/A",
          "to": "null"
        }
      ],
      "sparkline": {
        "fillColor": "rgba(31, 118, 189, 0.18)",
        "full": false,
        "lineColor": "rgb(31, 120, 193)",
        "show": false,
        "ymax": null,
        "ymin": null
      },
      "tableColumn": "",
      "targets": [
        {
          "expr": "(sum(increase(limited_calls[1m])) > 0 or vector(0))/(sum(increase(authorized_calls[1m]) or vector(0)) + sum(increase(limited_calls[1m]) or vector(0)))*100 > 0 or vector(0)",
          "instant": true,
          "refId": "A"
        }
      ],
      "thresholds": "",
      "timeFrom": null,
      "timeShift": null,
      "title": "Last 1 Minute - Rejected/Requests",
      "type": "singlestat",
      "valueFontSize": "80%",
      "valueMaps": [
        {
          "op": "=",
          "text": "N/A",
          "value": "null"
        }
      ],
      "valueName": "avg"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": null,
      "decimals": 0,
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
        "dataLinks": []
      },
      "percentage": false,
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "expr": "sum(increase(authorized_calls[1m]) or vector(0)) + sum(increase(limited_calls[1m]) or vector(0))",
          "instant": false,
          "interval": "30s",
          "legendFormat": "No. of Requests",
          "refId": "A"
        },
		{
          "expr": "$perMinuteRequestsPerUnit",
          "instant": false,
          "interval": "30s",
          "legendFormat": "Active Quota - ` + activeQuota + ` Per Day - Rate Limit - ` + requestsPerUnit + ` per minute",
          "refId": "B"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Per Minute API Requests",
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
        "#299c46",
        "rgba(237, 129, 40, 0.89)",
        "#d44a3a"
      ],
      "datasource": "Prometheus",
      "description": "",
      "format": "none",
      "gauge": {
        "maxValue": 100,
        "minValue": 0,
        "show": false,
        "thresholdLabels": false,
        "thresholdMarkers": true
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
      "options": {},
      "postfix": "",
      "postfixFontSize": "50%",
      "prefix": "",
      "prefixFontSize": "50%",
      "rangeMaps": [
        {
          "from": "null",
          "text": "N/A",
          "to": "null"
        }
      ],
      "sparkline": {
        "fillColor": "rgba(31, 118, 189, 0.18)",
        "full": false,
        "lineColor": "rgb(31, 120, 193)",
        "show": false,
        "ymax": null,
        "ymin": null
      },
      "tableColumn": "",
      "targets": [
        {
          "expr": "sum(increase(authorized_calls[24h]) or vector(0)) + sum(increase(limited_calls[24h]) or vector(0)) > 0 or vector(0)",
          "instant": true,
          "refId": "A"
        }
      ],
      "thresholds": "$perMinuteRequestsPerUnit*60*24",
      "timeFrom": null,
      "timeShift": null,
      "title": "Last 24 Hours - No. Requests",
      "type": "singlestat",
      "valueFontSize": "80%",
      "valueMaps": [
        {
          "op": "=",
          "text": "N/A",
          "value": "null"
        }
      ],
      "valueName": "avg"
    },
    {
      "cacheTimeout": null,
      "colorBackground": true,
      "colorValue": false,
      "colors": [
        "#299c46",
        "rgba(237, 129, 40, 0.89)",
        "#d44a3a"
      ],
      "datasource": "Prometheus",
      "format": "none",
      "gauge": {
        "maxValue": 100,
        "minValue": 0,
        "show": false,
        "thresholdLabels": false,
        "thresholdMarkers": true
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
      "options": {},
      "postfix": "",
      "postfixFontSize": "50%",
      "prefix": "",
      "prefixFontSize": "50%",
      "rangeMaps": [
        {
          "from": "null",
          "text": "N/A",
          "to": "null"
        }
      ],
      "sparkline": {
        "fillColor": "rgba(31, 118, 189, 0.18)",
        "full": false,
        "lineColor": "rgb(31, 120, 193)",
        "show": false,
        "ymax": null,
        "ymin": null
      },
      "tableColumn": "",
      "targets": [
        {
          "expr": "sum(increase(limited_calls[24h]) or vector(0))",
          "format": "time_series",
          "instant": true,
          "refId": "A"
        }
      ],
      "thresholds": "1",
      "timeFrom": null,
      "timeShift": null,
      "title": "Last 24 Hours - Rejected",
      "type": "singlestat",
      "valueFontSize": "80%",
      "valueMaps": [
        {
          "op": "=",
          "text": "N/A",
          "value": "null"
        }
      ],
      "valueName": "first"
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
      "description": "",
      "format": "none",
      "gauge": {
        "maxValue": 100,
        "minValue": 0,
        "show": false,
        "thresholdLabels": false,
        "thresholdMarkers": true
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
      "options": {},
      "postfix": "%",
      "postfixFontSize": "50%",
      "prefix": "",
      "prefixFontSize": "50%",
      "rangeMaps": [
        {
          "from": "null",
          "text": "N/A",
          "to": "null"
        }
      ],
      "sparkline": {
        "fillColor": "rgba(31, 118, 189, 0.18)",
        "full": false,
        "lineColor": "rgb(31, 120, 193)",
        "show": false,
        "ymax": null,
        "ymin": null
      },
      "tableColumn": "",
      "targets": [
        {
          "expr": "(sum(increase(limited_calls[24h])) > 0 or vector(0))/(sum(increase(authorized_calls[24h]) or vector(0)) + sum(increase(limited_calls[24h]) or vector(0)))*100 > 0 or vector(0)",
          "instant": true,
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "thresholds": "",
      "timeFrom": null,
      "timeShift": null,
      "title": "Last 24 Hours -  Rejected/Requests",
      "type": "singlestat",
      "valueFontSize": "80%",
      "valueMaps": [
        {
          "op": "=",
          "text": "N/A",
          "value": "null"
        }
      ],
      "valueName": "avg"
    }
  ],
  "refresh": "1m",
  "schemaVersion": 21,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "current": {
          "selected": false,
          "text": "` + requestsPerUnit + `",
          "value": "` + requestsPerUnit + `"
        },
        "hide": 2,
        "label": null,
        "name": "perMinuteRequestsPerUnit",
        "options": [
          {
            "selected": true,
            "text": "` + requestsPerUnit + `",
            "value": "` + requestsPerUnit + `"
          }
        ],
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
  "version": 1
}`
}

// The UID above is used to construct the url for the grafana dashboard in customer alerts. Please do not edit this value.
