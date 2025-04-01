package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "soa-project/post-service/proto"
	"soa-project/post-service/storage"
	shared "soa-project/shared/proto"
)

type PostService struct {
	pb.PostServiceServer
	storage *storage.Storage
}

func (s PostService) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Printf("%v vs %v", req.UserId, req.Post.AuthorId)
	if req.UserId.String() != req.Post.AuthorId.String() {
		return nil, status.Error(codes.PermissionDenied, "author does not much request issuer")
	}

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	postId, err := uuid.NewRandom()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate uuid: %v", err)
	}

	userId, err := uuid.Parse(req.UserId.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed id")
	}

	time := time.Now()

	post := storage.Post{
		PostId:         postId,
		Contents:       req.Post.Contents,
		Description:    req.Post.Description,
		AuthorId:       userId,
		Tags:           req.Post.Tags,
		IsPrivate:      req.Post.IsPrivate,
		CreationTime:   &time,
		LastUpdateTime: &time,
	}

	err = tx.InsertPost(ctx, post)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert post: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit tx: %v", err)
	}

	return &pb.CreatePostResponse{Id: &shared.Id{Uuid: postId.String()}}, nil
}

func (s PostService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	postId, err := uuid.Parse(req.Id.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed id")
	}
	userId, err := uuid.Parse(req.UserId.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed userId")
	}

	post, err := tx.FindPostByPostId(ctx, postId)
	if err == storage.ErrNoSuchPost {
		return nil, status.Error(codes.NotFound, "no post with provided id was found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not find post: %v", err)
	}

	if post.AuthorId != userId {
		return nil, status.Errorf(codes.PermissionDenied, "only author can delete own posts")
	}

	ok, err := tx.DeletePost(ctx, postId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete post: %v", err)
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "no post with provided id was found")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit tx: %v", err)
	}

	return &pb.DeletePostResponse{}, nil
}

func (s PostService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.UpdatePostResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	postId, err := uuid.Parse(req.Id.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed id")
	}
	userId, err := uuid.Parse(req.UserId.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed userId")
	}

	post, err := tx.FindPostByPostId(ctx, postId)
	if err == storage.ErrNoSuchPost {
		return nil, status.Error(codes.NotFound, "no post with provided id was found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not find post: %v", err)
	}

	if post.AuthorId != userId {
		return nil, status.Errorf(codes.PermissionDenied, "no rights to delete post")
	}

	time := time.Now()

	newPost := storage.Post{
		PostId:         post.PostId,
		Contents:       req.Post.Contents,
		Description:    req.Post.Description,
		AuthorId:       post.AuthorId,
		Tags:           req.Post.Tags,
		IsPrivate:      req.Post.IsPrivate,
		CreationTime:   post.CreationTime,
		LastUpdateTime: &time,
	}

	err = tx.UpdatePost(ctx, newPost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not update post: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit tx: %v", err)
	}

	return &pb.UpdatePostResponse{}, nil
}

func (s PostService) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	postId, err := uuid.Parse(req.Id.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed id")
	}
	userId, err := uuid.Parse(req.UserId.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed userId")
	}

	post, err := tx.FindPostByPostId(ctx, postId)
	if err == storage.ErrNoSuchPost {
		return nil, status.Error(codes.NotFound, "no post with provided id was found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not find post: %v", err)
	}

	if post.IsPrivate && post.AuthorId != userId {
		return nil, status.Error(codes.PermissionDenied, "no rights to view post")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit tx: %v", err)
	}

	return &pb.GetPostResponse{
		Post: &pb.Post{
			Contents:    post.Contents,
			Description: post.Description,
			AuthorId:    &shared.Id{Uuid: post.AuthorId.String()},
			Tags:        post.Tags,
			IsPrivate:   post.IsPrivate,
		},
		Meta: &pb.PostMetaInfo{
			CreationTime:   timestamppb.New(*post.CreationTime),
			LastUpdateTime: timestamppb.New(*post.LastUpdateTime),
		},
	}, nil
}

func (s PostService) GetPosts(ctx context.Context, req *pb.GetPostsRequest) (*pb.GetPostsResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := s.storage.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	userId, err := uuid.Parse(req.UserId.Uuid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to parse passed userId")
	}

	posts, err := tx.GetPageOfVisiblePosts(ctx, userId, req.PageNum, 2)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get page of posts: %v", err)
	}

	var postsId []*shared.Id
	var postsData []*pb.Post
	var postsMeta []*pb.PostMetaInfo
	for _, post := range posts {
		postsId = append(postsId, &shared.Id{Uuid: post.PostId.String()})
		postsData = append(postsData, &pb.Post{
			Contents:    post.Contents,
			Description: post.Description,
			AuthorId:    &shared.Id{Uuid: post.AuthorId.String()},
			Tags:        post.Tags,
			IsPrivate:   post.IsPrivate,
		})
		postsMeta = append(postsMeta, &pb.PostMetaInfo{
			CreationTime:   timestamppb.New(*post.CreationTime),
			LastUpdateTime: timestamppb.New(*post.LastUpdateTime),
		})
	}

	return &pb.GetPostsResponse{
		Ids:   postsId,
		Posts: postsData,
		Metas: postsMeta,
	}, nil
}

func NewUserService(databaseUrl string) (*PostService, error) {
	storage, err := storage.NewStorage(databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &PostService{
		storage: storage,
	}, nil
}
