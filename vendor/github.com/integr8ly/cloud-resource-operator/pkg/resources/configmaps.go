package resources

import (
	"context"
	"github.com/sirupsen/logrus"

	errorUtil "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetConfigMapOrDefault(ctx context.Context, c client.Client, name types.NamespacedName, def *v1.ConfigMap) (*v1.ConfigMap, error) {
	cm := &v1.ConfigMap{}
	if err := c.Get(ctx, name, cm); err != nil {
		if errors.IsNotFound(err) {
			logrus.Debugf("%s config not found in ns ( %s ) falling back to default strategy", name.Name, name.Namespace)
			return def, nil
		}
		return nil, errorUtil.Wrap(err, "failed to get config map, not returning default")
	}
	return cm, nil
}
