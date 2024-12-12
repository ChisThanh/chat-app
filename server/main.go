package main

import (
	"chat-app/proto/authpb"
	"chat-app/proto/chatpb"
	"chat-app/server/gRPC"
	"chat-app/server/utils"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var DB *mongo.Database

type ServerAuth struct {
	authpb.UnimplementedAuthServiceServer
}

type ServerChat struct {
	chatpb.UnimplementedChatServiceServer
}

type User struct {
	Name     string `bson:"name"`
	Email    string `bson:"email"`
	Password string `bson:"password"`
}

type MessageItem struct {
	UserID   string     `bson:"user_id"`
	Message  string     `bson:"message"`
	CreateAt *time.Time `bson:"created_at"`
}

type Chat struct {
	ChatID   string        `bson:"chat_id"`
	Messages []MessageItem `bson:"messages"`
}

func (s *ServerAuth) Login(ctx context.Context, in *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	log.Printf("Login request received for email: %s", in.GetEmail())
	var user User

	err := DB.Collection("users").FindOne(ctx, bson.M{"email": in.GetEmail()}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.GetPassword()))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, _ := utils.GenerateAccessToken(in.GetEmail())
	refreshToken, _ := utils.GenerateRefreshToken(in.GetEmail())

	return &authpb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *ServerAuth) Register(ctx context.Context, in *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	var existingUser User

	err := DB.Collection("users").FindOne(ctx, bson.M{"email": in.GetEmail()}).Decode(&existingUser)
	if err == nil {
		return nil, fmt.Errorf("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(in.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := User{
		Name:     in.GetName(),
		Email:    in.GetEmail(),
		Password: string(hashedPassword),
	}

	_, err = DB.Collection("users").InsertOne(ctx, newUser)
	if err != nil {
		return nil, err
	}

	accessToken, _ := utils.GenerateAccessToken(in.GetEmail())
	refreshToken, _ := utils.GenerateRefreshToken(in.GetEmail())

	return &authpb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func SaveMessageToDB(ctx context.Context, chatId string, userId string, message string) error {
	now := time.Now()
	messageItem := MessageItem{
		UserID:   userId,
		Message:  message,
		CreateAt: &now,
	}

	collection := DB.Collection("chat")
	filter := bson.M{"chat_id": chatId}

	var existingChat Chat
	err := collection.FindOne(ctx, filter).Decode(&existingChat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			chat := Chat{
				ChatID:   chatId,
				Messages: []MessageItem{messageItem},
			}
			_, err := collection.InsertOne(ctx, chat)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		update := bson.M{
			"$push": bson.M{
				"messages": messageItem,
			},
		}

		_, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ServerChat) SendMessage(ctx context.Context, in *chatpb.SendMessageRequest) (*chatpb.SendMessageResponse, error) {
	err := SaveMessageToDB(ctx, in.GetChatId(), in.GetUserId(), in.GetMessage())
	if err != nil {
		return nil, err
	}

	return &chatpb.SendMessageResponse{
		Message: in.GetMessage(),
		Status:  "Message sent successfully",
	}, nil
}

func (s *ServerChat) Connect(stream chatpb.ChatService_ConnectServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = stream.Send(&chatpb.ConnectResponse{
			ChatId:  req.GetChatId(),
			UserId:  req.GetUserId(),
			Message: fmt.Sprintf("Welcome, %s! Connected to chat %s.", req.GetUserId(), req.GetChatId()),
		})

		if err != nil {
			return err
		}
		err = SaveMessageToDB(stream.Context(),
			req.GetChatId(),
			req.GetUserId(),
			fmt.Sprintf("Welcome, %s! Connected to chat %s.", req.GetUserId(), req.GetChatId()))

		if err != nil {
			return err
		}
	}
}

func init() {
	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI("mongodb://admin:admin123@mongo:27017/")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	DB = client.Database("chat-app")
	log.Println("Connected to MongoDB!")
}

func main() {
	address := ":50051"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", address, err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(gRPC.UnaryInterceptor),
		grpc.StreamInterceptor(gRPC.StreamInterceptor),
	}

	sgRPC := grpc.NewServer(opts...)
	authpb.RegisterAuthServiceServer(sgRPC, &ServerAuth{})
	chatpb.RegisterChatServiceServer(sgRPC, &ServerChat{})
	reflection.Register(sgRPC)

	wrappedServer := grpcweb.WrapServer(sgRPC)

	httpServer := http.Server{
		Addr: ":50051",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if wrappedServer.IsGrpcWebRequest(r) || wrappedServer.IsAcceptableGrpcCorsRequest(r) {
				wrappedServer.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	}

	log.Printf("Starting HTTP server on %v", httpServer)
	log.Printf("Starting gRPC server on %s", address)

	if err := sgRPC.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
