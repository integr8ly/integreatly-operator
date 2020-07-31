package aws

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

//go:generate moq -out connection_tester_moq.go . ConnectionTester
type ConnectionTester interface {
	TCPConnection(host string, port int) bool
}

var _ ConnectionTester = (*ConnectionTestManager)(nil)

type ConnectionTestManager struct{}

func NewConnectionTestManager() *ConnectionTestManager {
	return &ConnectionTestManager{}
}

// TCPConnection trys to create a tcp connection, if none can be made it returns an error
func (m *ConnectionTestManager) TCPConnection(host string, port int) bool {
	logrus.Info(fmt.Sprintf("testing connectivity to host: %s", host))

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 10*time.Second)
	if err != nil {
		logrus.Error(fmt.Sprintf("tcp connection check failed for %s. reason : %s", host, err.Error()))
		return false
	}

	if err := conn.Close(); err != nil {
		logrus.Error(fmt.Sprintf("tcp connection failed to close to host: %s", host))
	}

	return true
}
