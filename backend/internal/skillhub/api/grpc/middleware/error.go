package middleware

import (
	"context"
	"errors"

	"github.com/fino-io/finokit/errorx"
	grpcx "github.com/fino-io/gokit/transport/grpc"
	"github.com/fino-io/skillhub/backend/pkg/errcode"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func errorMappingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		return resp, mapServerError(err)
	}
}

func mapServerError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	if business, ok := errorx.From(err); ok {
		code := codes.Internal
		// Add service-specific business error mappings here; unmapped business errors stay Internal.
		switch {
		case errcode.IsPlatformServiceUnavailable(business):
			code = codes.Unavailable
		case errcode.IsPlatformGatewayTimeout(business):
			code = codes.DeadlineExceeded
		case errcode.IsPlatformUnauthenticated(business):
			code = codes.Unauthenticated
		case errcode.IsPlatformForbidden(business):
			code = codes.PermissionDenied
		case errcode.IsPlatformNotFound(business), errcode.IsSkillhubNotFound(business):
			code = codes.NotFound
		case errcode.IsPlatformConflict(business):
			code = codes.AlreadyExists
		case errcode.IsPlatformTooManyRequests(business):
			code = codes.ResourceExhausted
		case errcode.IsPlatformBadRequest(business),
			errcode.IsSkillhubRequestMissing(business),
			errcode.IsSkillhubRequestValidationFailed(business),
			errcode.IsCreateSkillhubInputInvalid(business),
			errcode.IsUpdateSkillhubInputInvalid(business),
			errcode.IsListSkillhubsBadPageToken(business):
			code = codes.InvalidArgument
		}
		return status.Error(code, business.Message())
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	if code, ok := grpcx.CodeFromCoreError(err); ok {
		return status.Error(code, "request failed")
	}
	return status.Error(codes.Internal, "internal server error")
}
