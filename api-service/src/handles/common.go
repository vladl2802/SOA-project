package handles

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	postservice "soa-project/post-service/proto"
	shared "soa-project/shared/proto"
	userservice "soa-project/user-service/proto"
)

type UserId = uuid.UUID

type PostId = uuid.UUID

type User struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func UserPbToStruct(u *shared.User) User {
	return User{
		Login:    u.Login,
		Email:    u.Email,
		Password: u.Password,
	}
}

type Profile struct {
	Name           string    `json:"name,omitempty"`
	Surname        string    `json:"surname,omitempty"`
	PhoneNumber    string    `json:"phone_number,omitempty"`
	Birthday       string    `json:"birthday,omitempty"`
	CreationTime   time.Time `json:"creation_time"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

func ProfilePbToStruct(p *shared.Profile) Profile {
	birthday := ""
	if p.Birthday != nil {
		date := time.Date(int(p.Birthday.Year), time.Month(p.Birthday.Month), int(p.Birthday.Day), 0, 0, 0, 0, time.UTC)
		birthday = date.Format("2006-01-02")
	}

	return Profile{
		Name:           p.Name,
		Surname:        p.Surname,
		PhoneNumber:    p.PhoneNumber,
		Birthday:       birthday,
		CreationTime:   p.CreationTime.AsTime(),
		LastUpdateTime: p.LastUpdateTime.AsTime(),
	}
}

func ProfileStructToPb(p Profile) (*shared.Profile, error) {
	var birthday *shared.Date = nil
	if p.Birthday != "" {
		parsedDate, err := time.Parse("2006-01-02", p.Birthday)
		if err != nil {
			return nil, fmt.Errorf("failed parsing date: %v", err)
		}
		year, month, day := parsedDate.Date()
		birthday = &shared.Date{Year: int32(year), Month: int32(month), Day: int32(day)}
	}
	return &shared.Profile{
		Name:           p.Name,
		Surname:        p.Surname,
		PhoneNumber:    p.PhoneNumber,
		Birthday:       birthday,
		CreationTime:   timestamppb.New(p.CreationTime),
		LastUpdateTime: timestamppb.New(p.LastUpdateTime),
	}, nil
}

type HandleContext struct {
	PostserviceClient postservice.PostServiceClient
	UserserviceClient userservice.UserServiceClient
	EventsClient      EventsClient
	JwtPublic         *rsa.PublicKey
}

type JwtClaims struct {
	UserId UserId
}

func (h *HandleContext) parseAndVerifyJwtToken(jwtToken string) (*JwtClaims, error) {
	type Claims struct {
		UserId string `json:"user_id"`
		jwt.RegisteredClaims
	}

	token, err := jwt.ParseWithClaims(jwtToken, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("parse jwt token: unexpected signing method: %v", t.Header["alg"])
		}
		return h.JwtPublic, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt token: failed to parse: %w", err)
	}

	claims, ok := token.Claims.(*Claims)

	if !ok || !token.Valid {
		return nil, errors.New("parse jwt token: invalid token or claims")
	}

	uuid, err := uuid.Parse(claims.UserId)
	if err != nil {
		return nil, fmt.Errorf("parse jwt token: invalid user id provided: %w", err)
	}

	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		return nil, fmt.Errorf("provided jwt token expired")
	}

	return &JwtClaims{UserId: UserId(uuid)}, nil
}

type HandlerFunc func(*gin.Context)
