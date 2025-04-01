package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNoSuchPost = errors.New("no post were found")

type Post struct {
	PostId         uuid.UUID
	Contents       string
	Description    string
	AuthorId       uuid.UUID
	Tags           []string
	IsPrivate      bool
	CreationTime   *time.Time
	LastUpdateTime *time.Time
}

func postsTableSchema() string {
	return `
CREATE TABLE IF NOT EXISTS Posts (
	postId UUID PRIMARY KEY,
	contents TEXT,
	description TEXT,
	authorId UUID,
	tags VARCHAR[],
	isPrivate bool,
	creationTime TIMESTAMP WITHOUT TIME ZONE,
	lastUpdateTime TIMESTAMP WITHOUT TIME ZONE
);`
}

func getPostFromRow(row pgx.Row) (*Post, error) {
	var post Post
	err := row.Scan(&post.PostId, &post.Contents, &post.Description, &post.AuthorId, &post.Tags, &post.IsPrivate, &post.CreationTime, &post.LastUpdateTime)
	if err == pgx.ErrNoRows {
		return nil, ErrNoSuchPost
	}
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (tx *Tx) InsertPost(ctx context.Context, post Post) error {
	query := "INSERT INTO Posts (postId, contents, description, authorId, tags, isPrivate, creationTime, lastUpdateTime) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT DO NOTHING"
	fmt.Println("insert", post.PostId)
	_, err := tx.tx.Exec(ctx, query, post.PostId, post.Contents, post.Description, post.AuthorId, post.Tags, post.IsPrivate, post.CreationTime, post.LastUpdateTime)
	return err
}

func (tx *Tx) DeletePost(ctx context.Context, postId uuid.UUID) (bool, error) {
	const query = `DELETE FROM Posts WHERE postId = $1`

	cmd, err := tx.tx.Exec(ctx, query, postId)
	if err != nil {
		return false, err
	}

	if cmd.RowsAffected() == 0 {
		return false, nil
	}

	return true, nil
}

func (tx *Tx) UpdatePost(ctx context.Context, post Post) error {
	query := "UPDATE Posts SET contents = $1, description = $2, authorId = $3, tags = $4, isPrivate = $5, creationTime = $6, lastUpdateTime = $7 WHERE postId = $8"
	_, err := tx.tx.Exec(ctx, query, post.Contents, post.Description, post.AuthorId, post.Tags, post.IsPrivate, post.CreationTime, post.LastUpdateTime, post.PostId)
	return err
}

func (tx *Tx) FindPostByPostId(ctx context.Context, postId uuid.UUID) (*Post, error) {
	query := "SELECT * FROM Posts WHERE postId = $1"
	fmt.Println("select where postId = ", postId)
	return getPostFromRow(tx.tx.QueryRow(ctx, query, postId))
}

func (tx *Tx) GetPageOfVisiblePosts(ctx context.Context, userId uuid.UUID, page uint32, pageSize uint32) ([]Post, error) {
	query := "SELECT * FROM Posts WHERE NOT isPrivate OR authorId = $1 LIMIT $2 OFFSET $3"
	rows, err := tx.tx.Query(ctx, query, userId, pageSize, page*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		post, err := getPostFromRow(rows)
		if err != nil {
			return nil, err
		}

		posts = append(posts, *post)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}
