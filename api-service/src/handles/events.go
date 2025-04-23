package handles

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	postViewsTopic    = "post_views"
	userRegisterTopic = "user_register"
	commentsTopic     = "comments"
	likesTopic        = "likes"
)

type EventsClient struct {
	postViewsWriter    *kafka.Writer
	userRegisterWriter *kafka.Writer
	commentsWriter     *kafka.Writer
	likesWriter        *kafka.Writer
}

type EventPostView struct {
	User UserId
	Post PostId
	Time time.Time
}

type EventUserRegister struct {
	User UserId
	Time time.Time
}

type EventNewComment struct {
	Author UserId
	Post   PostId
	Time   time.Time
}

type EventNewLike struct {
	Author UserId
	Post   PostId
	Time   time.Time
}

func (e *EventsClient) OnPostView(ctx context.Context, event EventPostView) error {
	msg, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return e.postViewsWriter.WriteMessages(ctx, kafka.Message{
		Value: msg,
		Time:  time.Now(),
	})
}

func (e *EventsClient) OnUserRegister(ctx context.Context, event EventUserRegister) error {
	msg, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return e.userRegisterWriter.WriteMessages(ctx, kafka.Message{
		Value: msg,
		Time:  time.Now(),
	})
}

func (e *EventsClient) OnNewComment(ctx context.Context, event EventNewComment) error {
	msg, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return e.commentsWriter.WriteMessages(ctx, kafka.Message{
		Value: msg,
		Time:  time.Now(),
	})
}

func (e *EventsClient) OnNewLike(ctx context.Context, event EventNewLike) error {
	msg, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return e.likesWriter.WriteMessages(ctx, kafka.Message{
		Value: msg,
		Time:  time.Now(),
	})
}

func (e *EventsClient) Close() error {
	if e := e.postViewsWriter.Close(); e != nil {
		return e
	}
	if e := e.userRegisterWriter.Close(); e != nil {
		return e
	}
	if e := e.commentsWriter.Close(); e != nil {
		return e
	}
	if e := e.likesWriter.Close(); e != nil {
		return e
	}
	return nil
}

func (h *HandleContext) HandleEvents(engine *gin.Engine) {
	engine.POST("/comment", gin.HandlerFunc(handleNewComment(h)))
	engine.POST("/like", gin.HandlerFunc(handleNewLike(h)))
}

func handleNewComment(h *HandleContext) HandlerFunc {
	type request struct {
		Id string `json:"id"`
	}
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		var request request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/new_comment: couldn't bind input to json: %v", err)})
			return
		}

		postId, err := uuid.Parse(request.Id)
		if err != nil {
			log.Printf("/new_comment: couldn't parse post id: %v", err)
			ctx.Status(500)
			return
		}

		jwtToken, err := ctx.Cookie("jwt")
		if err != nil {
			ctx.JSON(401, map[string]any{"error": "/new_comment: missing jwt cookie"})
			return
		}

		claims, err := h.parseAndVerifyJwtToken(jwtToken)
		if err != nil {
			ctx.JSON(401, map[string]any{"error": fmt.Sprintf("/new_comment: jwt verification failed: %v", err)})
			return
		}

		err = h.EventsClient.OnNewComment(c, EventNewComment{
			Author: claims.UserId,
			Post:   PostId(postId),
			Time:   time.Now(),
		})
		if err != nil {
			log.Printf("/new_comment: couldn't send new comment event: %v", err)
			ctx.Status(500)
			return
		}

		ctx.Status(200)
	}
}

func handleNewLike(h *HandleContext) HandlerFunc {
	type request struct {
		Id string `json:"id"`
	}
	return func(ctx *gin.Context) {
		c, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		var request request
		err := ctx.ShouldBindJSON(&request)
		if err != nil {
			ctx.JSON(400, map[string]any{"error": fmt.Sprintf("/new_like: couldn't bind input to json: %v", err)})
			return
		}

		postId, err := uuid.Parse(request.Id)
		if err != nil {
			log.Printf("/new_like: couldn't parse post id: %v", err)
			ctx.Status(500)
			return
		}

		jwtToken, err := ctx.Cookie("jwt")
		if err != nil {
			ctx.JSON(401, map[string]any{"error": "/new_like: missing jwt cookie"})
			return
		}

		claims, err := h.parseAndVerifyJwtToken(jwtToken)
		if err != nil {
			ctx.JSON(401, map[string]any{"error": fmt.Sprintf("/new_like: jwt verification failed: %v", err)})
			return
		}

		err = h.EventsClient.OnNewLike(c, EventNewLike{
			Author: claims.UserId,
			Post:   PostId(postId),
			Time:   time.Now(),
		})
		if err != nil {
			log.Printf("/new_like: couldn't send post new like event: %v", err)
			ctx.Status(500)
			return
		}

		ctx.Status(200)
	}
}

func NewEventsClient(kafkaAddr string) EventsClient {
	return EventsClient{
		postViewsWriter: &kafka.Writer{
			Addr:                   kafka.TCP(kafkaAddr),
			Topic:                  postViewsTopic,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.LeastBytes{},
		},
		userRegisterWriter: &kafka.Writer{
			Addr:                   kafka.TCP(kafkaAddr),
			Topic:                  userRegisterTopic,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.LeastBytes{},
		},
		commentsWriter: &kafka.Writer{
			Addr:                   kafka.TCP(kafkaAddr),
			Topic:                  commentsTopic,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.LeastBytes{},
		},
		likesWriter: &kafka.Writer{
			Addr:                   kafka.TCP(kafkaAddr),
			Topic:                  likesTopic,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.LeastBytes{},
		},
	}
}
