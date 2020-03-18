package functional

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/integr8ly/integreatly-operator/test/common"
	"github.com/sirupsen/logrus"
)

type Client struct {
	ResourceManagers []ClusterResourceManager
	Logger           *logrus.Entry
}

func NewDefaultClient(awsSession *session.Session, logger *logrus.Entry) *Client {
	log := logger.WithField("cluster_service_provider", "aws")
	rdsManager := NewDefaultRDSInstanceManager(awsSession, logger)
	return &Client{
		ResourceManagers: []ClusterResourceManager{rdsManager},
		Logger:           log,
	}
}

//DeleteResourcesForCluster Delete AWS resources based on tags using provided action engines
func (c *Client) GetResourcesForCluster(clusterID string, tags map[string]string) (*ResourceCollection, error) {
	logger := c.Logger.WithFields(logrus.Fields{loggingKeyClusterID: clusterID})
	logger.Debugf("getting resources for cluster")
	resCollection := &ResourceCollection{}
	for _, engine := range c.ResourceManagers {
		engineLogger := logger.WithField(loggingKeyManager, engine.GetName())
		engineLogger.Debugf("found Logger")
		output, err := engine.GetResourcesForCluster(clusterID, tags)
		if err != nil {
			return nil, common.WrapLog(err, fmt.Sprintf("failed to run engine %s", engine.GetName()), engineLogger)
		}
		resCollection.Resources = append(resCollection.Resources, output...)

	}
	return resCollection, nil
}
