package handles

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	postservice "soa-project/post-service/proto"
	shared "soa-project/shared/proto"
)

func (h *HandleContext) HandlePostService(engine *gin.Engine) {
	engine.POST("/post", gin.HandlerFunc(handleCreatePost(h)))
	engine.POST("/post/delete", gin.HandlerFunc(handleDeletePost(h)))
	engine.POST("/post/update", gin.HandlerFunc(handleUpdatePost(h)))
	engine.GET("/post", gin.HandlerFunc(handleGetPost(h)))
	engine.GET("/posts", gin.HandlerFunc(handleGetPosts(h)))
}

func handleCreatePost(h *HandleContext) HandlerFunc {
	type request struct {
		Contents    string   `json:"contents"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		IsPrivate   bool     `json:"is_private"`
	}

	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		var request request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/register: couldn't bind input to json: %v", err)})
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

		response, err := h.PostserviceClient.CreatePost(c, &postservice.CreatePostRequest{
			UserId: &shared.Id{Uuid: claims.UserId.String()},
			Post: &postservice.Post{
				Contents:    request.Contents,
				Description: request.Description,
				AuthorId:    &shared.Id{Uuid: claims.UserId.String()},
				Tags:        request.Tags,
				IsPrivate:   request.IsPrivate,
			},
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/post: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.PermissionDenied:
				ctx.JSON(409, map[string]any{"error": fmt.Sprintf("/post: permission denied: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/post: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/post: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		ctx.JSON(201, map[string]any{"post_id": response.Id.Uuid})
	}
}

func handleDeletePost(h *HandleContext) HandlerFunc {
	type request struct {
		Id string `json:"id"`
	}

	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		var request request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/register: couldn't bind input to json: %v", err)})
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

		_, err = h.PostserviceClient.DeletePost(c, &postservice.DeletePostRequest{
			UserId: &shared.Id{Uuid: claims.UserId.String()},
			Id:     &shared.Id{Uuid: request.Id},
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/post/delete: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.NotFound:
				ctx.JSON(404, map[string]any{"error": fmt.Sprintf("/post/update: %v", st.Err().Error())})
			case codes.PermissionDenied:
				ctx.JSON(409, map[string]any{"error": fmt.Sprintf("/post/delete: permission denied: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/post/delete: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/post/delete: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		ctx.Status(200)
	}
}

func handleUpdatePost(h *HandleContext) HandlerFunc {
	type request struct {
		Contents    string   `json:"contents"`
		Description string   `json:"description"`
		Id          string   `json:"id"`
		Tags        []string `json:"tags"`
		IsPrivate   bool     `json:"is_private"`
	}

	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		var request request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/post/update: couldn't bind input to json: %v", err)})
			return
		}

		jwtToken, err := ctx.Cookie("jwt")
		if err != nil {
			ctx.JSON(401, map[string]any{"error": "/post/update: missing jwt cookie"})
			return
		}

		claims, err := h.parseAndVerifyJwtToken(jwtToken)
		if err != nil {
			ctx.JSON(401, map[string]any{"error": fmt.Sprintf("/post/update: jwt verification failed: %v", err)})
			return
		}

		_, err = h.PostserviceClient.UpdatePost(c, &postservice.UpdatePostRequest{
			UserId: &shared.Id{Uuid: claims.UserId.String()},
			Id:     &shared.Id{Uuid: request.Id},
			Post: &postservice.Post{
				Contents:    request.Contents,
				Description: request.Description,
				AuthorId:    &shared.Id{Uuid: claims.UserId.String()},
				Tags:        request.Tags,
				IsPrivate:   request.IsPrivate,
			},
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/post/update: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.NotFound:
				ctx.JSON(404, map[string]any{"error": fmt.Sprintf("/post/update: %v", st.Err().Error())})
			case codes.PermissionDenied:
				ctx.JSON(409, map[string]any{"error": fmt.Sprintf("/post/update: permission denied: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/post/update: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/post/update: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		ctx.Status(200)
	}
}

func handleGetPost(h *HandleContext) HandlerFunc {
	type request struct {
		Id     string `json:"id"`
		UserId string `json:"user_id"`
	}

	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		var request request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/post: couldn't bind input to json: %v", err)})
			return
		}

		jwtToken, err := ctx.Cookie("jwt")
		if err != nil {
			ctx.JSON(401, map[string]any{"error": "/post: missing jwt cookie"})
			return
		}

		claims, err := h.parseAndVerifyJwtToken(jwtToken)
		if err != nil {
			ctx.JSON(401, map[string]any{"error": fmt.Sprintf("/post: jwt verification failed: %v", err)})
			return
		}

		response, err := h.PostserviceClient.GetPost(c, &postservice.GetPostRequest{
			UserId: &shared.Id{Uuid: claims.UserId.String()},
			Id:     &shared.Id{Uuid: request.Id},
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/post: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.NotFound:
				ctx.JSON(404, map[string]any{"error": fmt.Sprintf("/post: %v", st.Err().Error())})
			case codes.PermissionDenied:
				ctx.JSON(409, map[string]any{"error": fmt.Sprintf("/post: permission denied: %v", st.Err().Error())})
			case codes.Internal:
				log.Printf("/post: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/post: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		ctx.JSON(200, map[string]any{
			"contents":         response.Post.Contents,
			"description":      response.Post.Description,
			"author_id":        response.Post.AuthorId,
			"tags":             response.Post.Tags,
			"is_private":       response.Post.IsPrivate,
			"creation_time":    response.Meta.CreationTime,
			"last_update_time": response.Meta.LastUpdateTime,
		})
	}
}

func handleGetPosts(h *HandleContext) HandlerFunc {
	type request struct {
		PageNum uint32 `json:"page_num"`
	}

	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		var request request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/posts: couldn't bind input to json: %v", err)})
			return
		}

		jwtToken, err := ctx.Cookie("jwt")
		if err != nil {
			ctx.JSON(401, map[string]any{"error": "/posts: missing jwt cookie"})
			return
		}

		claims, err := h.parseAndVerifyJwtToken(jwtToken)
		if err != nil {
			ctx.JSON(401, map[string]any{"error": fmt.Sprintf("/posts: jwt verification failed: %v", err)})
			return
		}

		response, err := h.PostserviceClient.GetPosts(c, &postservice.GetPostsRequest{
			UserId:  &shared.Id{Uuid: claims.UserId.String()},
			PageNum: request.PageNum,
		})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok {
				log.Printf("/posts: grpc err: %v\n", err)
				ctx.Status(500)
				return
			}
			switch st.Code() {
			case codes.Internal:
				log.Printf("/posts: internal error: %v\n", st.Err().Error())
				ctx.Status(500)
			default:
				log.Printf("/posts: non recognized status: %v (code: %v)\n", st.Err().Error(), st.Code())
				ctx.Status(500)
			}
			return
		}

		var result []map[string]any

		for i := range len(response.Metas) {
			id := response.Ids[i]
			post := response.Posts[i]
			meta := response.Metas[i]
			result = append(result, map[string]any{
				"id":               id.String(),
				"contents":         post.Contents,
				"description":      post.Description,
				"author_id":        post.AuthorId.String(),
				"tags":             post.Tags,
				"is_private":       post.IsPrivate,
				"creation_time":    meta.CreationTime,
				"last_update_time": meta.LastUpdateTime,
			})
		}

		ctx.JSON(200, result)
	}
}
