package main

import (
	"context"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"os"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	shared "soa-project/shared/proto"
	pb "soa-project/user-service/proto"
	"soa-project/user-service/storage"
)

type JwtManager struct {
	jwtPrivate *rsa.PrivateKey
}

type UserService struct {
	pb.UnimplementedUserServiceServer
	storage    *storage.Storage
	jwtManager JwtManager
}

func checkLoginCorrectness(login string) error {
	if len(login) > 100 {
		return errors.New("too long login")
	}

	if len(login) == 0 {
		return errors.New("empty login")
	}

	if !utf8.ValidString(login) {
		return errors.New("not valid utf8")
	}

	for _, r := range login {
		if unicode.IsSpace(r) {
			return errors.New("login must not contain spaces")
		}
	}

	return nil
}

func checkEmailCorrectness(email string) error {
	if len(email) > 255 {
		return errors.New("too long email address")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	return nil
}

func (s UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	err = checkLoginCorrectness(req.Login)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid login: %v", err)
	}
	_, err = tx.FindUserByLogin(ctx, req.Login)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "login is already used")
	} else if err != storage.ErrNoSuchUser {
		return nil, status.Errorf(codes.Internal, "failed to find user by login: %v", err)
	}

	err = checkEmailCorrectness(req.Email)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email address: %v", err)
	}
	_, err = tx.FindUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "email is already used")
	} else if err != storage.ErrNoSuchUser {
		return nil, status.Errorf(codes.Internal, "failed to find user by email: %v", err)
	}

	userId, err := uuid.NewRandom()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate uuid: %v", err)
	}
	preHashedPassword, err := hex.DecodeString(req.HashedPassword)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid hex hashed password provided")
	}

	// password checking is done on the apiGateway side
	hashedPass, err := bcrypt.GenerateFromPassword(preHashedPassword, bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to bcrypt hashed password: %v", err)
	}

	user := storage.User{
		UserId:         userId,
		Login:          req.Login,
		Email:          req.Email,
		HashedPassword: hashedPass,
	}

	time := time.Now()

	profile := storage.Profile{
		UserId:         userId,
		CreationTime:   &time,
		LastUpdateTime: &time,
	}

	err = tx.InsertUser(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert user: %v", err)
	}

	err = tx.InsertProfile(ctx, profile)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert profile: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit tx: %v", err)
	}

	return &pb.RegisterResponse{Id: &shared.Id{Uuid: userId.String()}}, nil
}

func (s UserService) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	var user *storage.User
	if req.Login != "" {
		err = checkLoginCorrectness(req.Login)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid email: %v", err)
		}

		user, err = tx.FindUserByLogin(ctx, req.Login)
		if err != nil && err != storage.ErrNoSuchUser {
			return nil, status.Errorf(codes.Internal, "failed to find user by login failed: %v", err)
		}
	}
	if req.Email != "" {
		err = checkEmailCorrectness(req.Email)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid login: %v", err)
		}

		user, err = tx.FindUserByEmail(ctx, req.Email)
		if err != nil && err != storage.ErrNoSuchUser {
			return nil, status.Errorf(codes.Internal, "failed to find user by email failed: %v", err)
		}
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "no user with such login/email were found")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit tx: %v", err)
	}

	preHashedPassword, err := hex.DecodeString(req.HashedPassword)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid hex hashed password provided")
	}

	err = bcrypt.CompareHashAndPassword(user.HashedPassword, preHashedPassword)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid password")
	}

	issuedTime := time.Now()
	expirationTime := issuedTime.Add(time.Minute * 10)
	type Claims struct {
		UserId string `json:"user_id"`
		jwt.RegisteredClaims
	}

	claims := Claims{
		user.UserId.String(),
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(issuedTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	value, err := token.SignedString(s.jwtManager.jwtPrivate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "jwt signing error: %v", err)
	}

	return &pb.AuthResponse{Id: &shared.Id{Uuid: user.UserId.String()}, Jwt: value}, nil
}

func (s UserService) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	userId, err := uuid.Parse(req.Id.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed id")
	}

	oldProfile, err := tx.FindProfileByUserId(ctx, userId)
	if err != nil {
		if err == storage.ErrNoSuchUser {
			return nil, status.Error(codes.NotFound, "no profile for provided used id")
		} else {
			return nil, status.Errorf(codes.Internal, "failed to find profile by userId: %v", err)
		}
	}

	reqProf := req.Profile
	var birthDate *time.Time
	if reqProf.Birthday != nil {
		date := time.Date(int(reqProf.Birthday.Year), time.Month(reqProf.Birthday.Month), int(reqProf.Birthday.Day), 0, 0, 0, 0, time.UTC)
		birthDate = &date
	}
	time := time.Now()

	profile := storage.Profile{
		UserId:         userId,
		Name:           reqProf.Name,
		Surname:        reqProf.Surname,
		PhoneNumber:    reqProf.PhoneNumber,
		BirthDay:       birthDate,
		CreationTime:   oldProfile.CreationTime,
		LastUpdateTime: &time,
	}

	log.Printf("profile: %v", profile)

	err = tx.UpdateProfile(ctx, profile)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update profile failed: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit tx: %v", err)
	}

	return &pb.UpdateProfileResponse{}, nil
}

func (s UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	userId, err := uuid.Parse(req.Id.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed id")
	}

	user, err := tx.FindUserById(ctx, userId)
	if err != nil {
		if err == storage.ErrNoSuchUser {
			return nil, status.Error(codes.NotFound, "no user for provided used id")
		} else {
			return nil, status.Errorf(codes.Internal, "failed to find user by userId: %v", err)
		}
	}

	return &pb.GetUserResponse{Login: string(user.Login[:]), Email: string(user.Email[:])}, nil
}

func (s UserService) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	userId, err := uuid.Parse(req.Id.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed id")
	}

	profile, err := tx.FindProfileByUserId(ctx, userId)
	if err != nil {
		if err == storage.ErrNoSuchUser {
			return nil, status.Error(codes.NotFound, "no profile for provided used id")
		} else {
			return nil, status.Errorf(codes.Internal, "failed to find profile by userId: %v", err)
		}
	}

	var birthday *shared.Date = nil
	if profile.BirthDay != nil {
		year, month, day := profile.BirthDay.Date()
		birthday = &shared.Date{Year: int32(year), Month: int32(month), Day: int32(day)}
	}

	respProfile := shared.Profile{
		Name:           profile.Name,
		Surname:        profile.Surname,
		PhoneNumber:    profile.PhoneNumber,
		Birthday:       birthday,
		CreationTime:   timestamppb.New(*profile.CreationTime),
		LastUpdateTime: timestamppb.New(*profile.LastUpdateTime),
	}

	return &pb.GetProfileResponse{Profile: &respProfile}, nil
}

func NewUserService(jwtPrivateFile string, databaseUrl string) (*UserService, error) {
	private, err := os.ReadFile(jwtPrivateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read jwtPrivateFile: %w", err)
	}
	jwtPrivate, err := jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	storage, err := storage.NewStorage(databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &UserService{
		storage: storage,
		jwtManager: JwtManager{
			jwtPrivate: jwtPrivate,
			// jwtPublic:  jwtPublic,
		},
	}, nil
}
