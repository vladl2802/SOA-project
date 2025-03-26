package handles

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	shared "soa-project/shared/proto"
	userservice "soa-project/user-service/proto"
)

// TODO: Clean up contexts. As for now they are pretty useless.

func (h *HandleContext) HandleUserService(engine *gin.Engine) {
	engine.POST("/register", gin.HandlerFunc(handleRegister(h)))
	engine.POST("/auth", gin.HandlerFunc(handleAuth(h)))
	engine.GET("/users", gin.HandlerFunc(handleGetUserById(h)))
	engine.GET("/profiles", gin.HandlerFunc(handleGetProfileById(h)))
	engine.POST("/profiles/update", gin.HandlerFunc(handleUpdateProfile(h)))
}

func handleRegister(h *HandleContext) HandlerFunc {
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		var user User
		err := ctx.ShouldBindJSON(&user)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/register: couldn't bind input to json: %v", err)})
			return
		}

		if err := checkPasswordValidity(user.Password); err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/register: password is invalid: %v", err)})
			return
		}

		hashedPass := hashPassword(user)

		response, err := h.UserserviceClient.Register(c, &userservice.RegisterRequest{
			Login:          user.Login,
			Email:          user.Email,
			HashedPassword: hex.EncodeToString(hashedPass[:]),
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/register: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.AlreadyExists:
				ctx.JSON(409, map[string]any{"error": "/register: user with provided login/email already exists"})
			case codes.InvalidArgument:
				ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/register: login/email have unexpected format: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/register: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/register: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		ctx.JSON(201, map[string]any{"user_id": response.Id.String()})
	}
}

func handleAuth(h *HandleContext) HandlerFunc {
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		var user User
		err := ctx.ShouldBindJSON(&user)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/auth: couldn't bind input to json: %v", err)})
			return
		}

		if err := checkPasswordValidity(user.Password); err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/auth: password is invalid: %v", err)})
			return
		}

		hashedPass := hashPassword(user)

		response, err := h.UserserviceClient.Auth(c, &userservice.AuthRequest{
			Login:          user.Login,
			Email:          user.Email,
			HashedPassword: hex.EncodeToString(hashedPass[:]),
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/auth: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.AlreadyExists:
				ctx.JSON(409, map[string]any{"error": "/auth: user with provided login/email already exists"})
			case codes.InvalidArgument:
				ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/auth: login/email have unexpected format or invalid password provided: %v", st.Err().Error())})
			case codes.NotFound:
				ctx.JSON(404, map[string]any{"error": fmt.Sprintf("/auth: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/auth: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/auth: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		ctx.JSON(200, map[string]any{"user_id": response.Id.String(), "jwt": response.Jwt})
	}
}

func handleGetUserById(h *HandleContext) HandlerFunc {
	type Request struct {
		Id string `json:"user_id"`
	}

	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		var request Request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/users: couldn't bind input to json: %v", err)})
			return
		}

		uuid, err := uuid.Parse(request.Id)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/users: couldn't retrieve id: %v", err)})
			return
		}

		response, err := h.UserserviceClient.GetUser(c, &userservice.GetUserRequest{
			Id: &shared.Id{Uuid: uuid.String()},
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/users: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.NotFound:
				ctx.JSON(404, map[string]any{"error": fmt.Sprintf("/users: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/users: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/users: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		ctx.JSON(200, map[string]any{"login": response.Login, "email": response.Email})
	}
}

func handleGetProfileById(h *HandleContext) HandlerFunc {
	type Request struct {
		Id string `json:"user_id"`
	}
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		var request Request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/profiles: couldn't bind input to json: %v", err)})
			return
		}

		uuid, err := uuid.Parse(request.Id)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/profiles: couldn't retrieve id: %v", err)})
			return
		}

		response, err := h.UserserviceClient.GetProfile(c, &userservice.GetProfileRequest{
			Id: &shared.Id{Uuid: uuid.String()},
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/profiles: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.NotFound:
				ctx.JSON(404, map[string]any{"error": fmt.Sprintf("/profiles: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/profiles: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/profiles: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		profile := response.Profile
		if profile == nil {
			log.Printf("/profiles: received empty profile")
			ctx.Status(500)
			return
		}

		ctx.JSON(200, ProfilePbToStruct(profile))
	}
}

func handleUpdateProfile(h *HandleContext) HandlerFunc {
	type Request struct {
		Id      string `json:"user_id"`
		Profile `json:",inline"`
	}

	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		var request Request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/profiles/update: couldn't bind input to json: %v", err)})
			return
		}

		log.Printf("update: request: %v", request)

		uuid, err := uuid.Parse(request.Id)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/profiles/update: couldn't retrieve id: %v", err)})
			return
		}

		jwtToken, err := ctx.Cookie("jwt")
		if err != nil {
			ctx.JSON(401, map[string]any{"error": "/profiles/update: missing jwt cookie"})
			return
		}

		claims, err := h.parseAndVerifyJwtToken(jwtToken)
		if err != nil {
			ctx.JSON(401, map[string]any{"error": fmt.Sprintf("/profiles/update: jwt verification failed: %v", err)})
			return
		}

		log.Printf("update: claims: %v", claims)

		if claims.UserId != uuid {
			ctx.JSON(401, map[string]any{"error": "/profiles/update: request issuer has no rights to perform this operation"})
			return
		}

		profile, err := ProfileStructToPb(request.Profile)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/profiles/update: provided profile is invalid: %v", err)})
			return
		}

		_, err = h.UserserviceClient.UpdateProfile(c, &userservice.UpdateProfileRequest{
			Id:      &shared.Id{Uuid: uuid.String()},
			Profile: profile,
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/profiles/update: grpc err: %v\n", err)
				ctx.Status(501)
				return
			}
			switch st.Code() {
			case codes.NotFound:
				ctx.JSON(404, map[string]any{"error": fmt.Sprintf("/profiles/update: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/profiles/update: internal error: %v\n", st.Err().Error())
				ctx.Status(502)
			default:
				log.Printf("/profiles/update: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(503)
			}
			return
		}

		ctx.Status(200)
	}
}

func hashPassword(user User) [16]byte {
	return md5.Sum([]byte(user.Password + user.Login))
}

func checkPasswordValidity(password string) error {
	if len(password) > 72 {
		return errors.New("too long password")
	}
	if len(password) < 7 {
		return errors.New("too short password")
	}

	controls := 0
	digits := 0
	letters := 0

	for _, r := range password {
		if r > unicode.MaxASCII {
			return errors.New("non-ASCII character in password")
		}
		switch {
		case unicode.IsDigit(r):
			digits++
		case unicode.IsLetter(r):
			letters++
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			controls++
		case unicode.IsSpace(r):
			return errors.New("space in password")
		}
	}

	if controls == 0 || digits == 0 || letters == 0 {
		return errors.New("password doesn't contain enough variety of symbols")
	}

	return nil
}
