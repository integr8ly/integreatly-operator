package gcpiface

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

type SQLAdminService interface {
	InstancesList(string) (*sqladmin.InstancesListResponse, error)
	DeleteInstance(context.Context, string, string) (*sqladmin.Operation, error)
	CreateInstance(context.Context, string, *sqladmin.DatabaseInstance) (*sqladmin.Operation, error)
	ModifyInstance(context.Context, string, string, *sqladmin.DatabaseInstance) (*sqladmin.Operation, error)
	GetInstance(context.Context, string, string) (*sqladmin.DatabaseInstance, error)
	ExportDatabase(ctx context.Context, project, instanceName string, req *sqladmin.InstancesExportRequest) (*sqladmin.Operation, error)
}

func NewSQLAdminService(ctx context.Context, opt option.ClientOption, logger *logrus.Entry) (SQLAdminService, error) {
	sqladminService, err := sqladmin.NewService(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &sqlClient{
		sqlAdminService: sqladminService,
		logger:          logger,
	}, nil

}

// wrapper for real client
type sqlClient struct {
	SQLAdminService
	sqlAdminService *sqladmin.Service
	logger          *logrus.Entry
}

func (r *sqlClient) InstancesList(project string) (*sqladmin.InstancesListResponse, error) {
	r.logger.Info("listing gcp postgres instances")
	return r.sqlAdminService.Instances.List(project).Do()
}

func (r *sqlClient) DeleteInstance(ctx context.Context, projectID, instanceName string) (*sqladmin.Operation, error) {
	r.logger.Infof("deleting gcp postgres instance %s", instanceName)
	return r.sqlAdminService.Instances.Delete(projectID, instanceName).Context(ctx).Do()
}

func (r *sqlClient) CreateInstance(ctx context.Context, projectID string, databaseInstance *sqladmin.DatabaseInstance) (*sqladmin.Operation, error) {
	r.logger.Infof("creating gcp postgres instance %s", databaseInstance.Name)
	return r.sqlAdminService.Instances.Insert(projectID, databaseInstance).Context(ctx).Do()
}

func (r *sqlClient) ModifyInstance(ctx context.Context, projectID, instanceName string, databaseInstance *sqladmin.DatabaseInstance) (*sqladmin.Operation, error) {
	r.logger.Infof("patching gcp postgres instance %s", databaseInstance.Name)
	return r.sqlAdminService.Instances.Patch(projectID, instanceName, databaseInstance).Context(ctx).Do()
}

func (r *sqlClient) GetInstance(ctx context.Context, projectID, instanceName string) (*sqladmin.DatabaseInstance, error) {
	r.logger.Infof("fetching gcp postgres instance %s", instanceName)
	return r.sqlAdminService.Instances.Get(projectID, instanceName).Context(ctx).Do()
}

func (r *sqlClient) ExportDatabase(ctx context.Context, projectID, instanceName string, req *sqladmin.InstancesExportRequest) (*sqladmin.Operation, error) {
	r.logger.Infof("exporting gcp postgres database from instance %s", instanceName)
	return r.sqlAdminService.Instances.Export(projectID, instanceName, req).Context(ctx).Do()
}

type MockSqlClient struct {
	SQLAdminService
	InstancesListFn  func(string) (*sqladmin.InstancesListResponse, error)
	DeleteInstanceFn func(context.Context, string, string) (*sqladmin.Operation, error)
	CreateInstanceFn func(context.Context, string, *sqladmin.DatabaseInstance) (*sqladmin.Operation, error)
	ModifyInstanceFn func(context.Context, string, string, *sqladmin.DatabaseInstance) (*sqladmin.Operation, error)
	GetInstanceFn    func(context.Context, string, string) (*sqladmin.DatabaseInstance, error)
	ExportDatabaseFn func(context.Context, string, string, *sqladmin.InstancesExportRequest) (*sqladmin.Operation, error)
}

func GetMockSQLClient(modifyFn func(sqlClient *MockSqlClient)) *MockSqlClient {
	mock := &MockSqlClient{
		InstancesListFn: func(project string) (*sqladmin.InstancesListResponse, error) {
			return &sqladmin.InstancesListResponse{
				Items: []*sqladmin.DatabaseInstance{},
			}, nil
		},
		DeleteInstanceFn: func(ctx context.Context, projectID, instanceName string) (*sqladmin.Operation, error) {
			return &sqladmin.Operation{}, nil
		},
		CreateInstanceFn: func(ctx context.Context, projectID string, databaseInstance *sqladmin.DatabaseInstance) (*sqladmin.Operation, error) {
			return &sqladmin.Operation{}, nil
		},
		ModifyInstanceFn: func(ctx context.Context, projectID, instanceName string, databaseInstance *sqladmin.DatabaseInstance) (*sqladmin.Operation, error) {
			return &sqladmin.Operation{}, nil
		},
		GetInstanceFn: func(ctx context.Context, projectID, instanceName string) (*sqladmin.DatabaseInstance, error) {
			return nil, nil
		},
		ExportDatabaseFn: func(ctx context.Context, projectID, instanceName string, req *sqladmin.InstancesExportRequest) (*sqladmin.Operation, error) {
			return &sqladmin.Operation{}, nil
		},
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func (m *MockSqlClient) InstancesList(project string) (*sqladmin.InstancesListResponse, error) {
	return m.InstancesListFn(project)
}

func (m *MockSqlClient) DeleteInstance(ctx context.Context, projectID, instanceName string) (*sqladmin.Operation, error) {
	return m.DeleteInstanceFn(ctx, projectID, instanceName)
}

func (m *MockSqlClient) CreateInstance(ctx context.Context, projectID string, instance *sqladmin.DatabaseInstance) (*sqladmin.Operation, error) {
	return m.CreateInstanceFn(ctx, projectID, instance)
}

func (m *MockSqlClient) ModifyInstance(ctx context.Context, projectID, instanceName string, instance *sqladmin.DatabaseInstance) (*sqladmin.Operation, error) {
	return m.ModifyInstanceFn(ctx, projectID, instance.Name, instance)
}

func (m *MockSqlClient) GetInstance(ctx context.Context, projectID, instanceName string) (*sqladmin.DatabaseInstance, error) {
	return m.GetInstanceFn(ctx, projectID, instanceName)
}

func (m *MockSqlClient) ExportDatabase(ctx context.Context, projectID, instanceName string, req *sqladmin.InstancesExportRequest) (*sqladmin.Operation, error) {
	return m.ExportDatabaseFn(ctx, projectID, instanceName, req)
}
