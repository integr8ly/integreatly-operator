package resources

import (
	"errors"
	"fmt"
	googleGRPC "github.com/googleapis/gax-go/v2/apierror"
	googleHTTP "google.golang.org/api/googleapi"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"net/http"
)

func IsNotFoundError(err error) bool {
	var googleHttpErr *googleHTTP.Error
	if errors.As(err, &googleHttpErr) {
		if googleHttpErr.Code == http.StatusNotFound {
			return true
		}
	}
	var googleGrpcErr *googleGRPC.APIError
	if errors.As(err, &googleGrpcErr) {
		if googleGrpcErr.GRPCStatus().Code() == grpcCodes.NotFound {
			return true
		}
	}
	var k8sErr *k8sErrors.StatusError
	if errors.As(err, &k8sErr) {
		if k8sErr.ErrStatus.Code == http.StatusNotFound {
			return true
		}
	}
	return false
}

type ErrorGRPC struct {
	grpcCode grpcCodes.Code
	message  string
}

func (e ErrorGRPC) Error() string {
	return fmt.Sprintf("mock ErrorGRPC.Error() called with code %d and message: %s", e.grpcCode, e.message)
}

func (e ErrorGRPC) GRPCStatus() *status.Status {
	return status.New(e.grpcCode, e.message)
}

// NewMockAPIError is used for mocking errors from package github.com/googleapis/gax-go/v2/apierror
func NewMockAPIError(grpcCode grpcCodes.Code) *googleGRPC.APIError {
	apiError, _ := googleGRPC.FromError(&ErrorGRPC{
		grpcCode: grpcCode,
		message:  "placeholder",
	})
	return apiError
}
