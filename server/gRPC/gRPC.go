package gRPC

import (
	"chat-app/server/utils"
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Common function to handle token validation
func validateToken(ctx context.Context, fullMethod string, excludedRoutes []string, refreshTokenRoute string) error {
	// Bỏ qua xác thực cho các route trong danh sách loại trừ
	for _, route := range excludedRoutes {
		if fullMethod == route {
			return nil
		}
	}

	// Lấy metadata từ context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("authorization token missing")
	}

	// Lấy token từ metadata
	tokens := md.Get("Authorization")
	if len(tokens) == 0 {
		return fmt.Errorf("authorization token missing")
	}

	// Cắt chuỗi "Bearer " để lấy token
	accessToken := strings.TrimPrefix(tokens[0], "Bearer ")
	typeToken := utils.CheckTokenType(accessToken)
	log.Printf("Type token: %s", typeToken)

	// Kiểm tra loại token
	switch typeToken {
	case "access_token":
		// Cho phép tiếp tục nếu là access token
		return nil
	case "refresh_token":
		// Cho phép tiếp tục chỉ nếu route là RefreshToken
		if fullMethod == refreshTokenRoute {
			return nil
		}
		return fmt.Errorf("invalid token type for this route")
	default:
		return fmt.Errorf("invalid token")
	}
}

// UnaryInterceptor implementation
func UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	excludedRoutes := []string{
		"/authpb.AuthService/Login",
		"/authpb.AuthService/Register",
	}
	refreshTokenRoute := "/authpb.AuthService/RefreshToken"

	// Validate token
	if err := validateToken(ctx, info.FullMethod, excludedRoutes, refreshTokenRoute); err != nil {
		return nil, err
	}

	// Continue with the handler
	return handler(ctx, req)
}

// StreamInterceptor implementation
func StreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	excludedRoutes := []string{
		"/authpb.AuthService/LoginStream",
		"/authpb.AuthService/RegisterStream",
	}
	refreshTokenRoute := "/authpb.AuthService/RefreshTokenStream"

	// Validate token
	if err := validateToken(ss.Context(), info.FullMethod, excludedRoutes, refreshTokenRoute); err != nil {
		return err
	}

	// Continue with the handler
	return handler(srv, ss)
}
