package infrastructure

import (
	"context"
	"fmt"

	"github.com/ecommerce/backend/internal/auth/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type tokenRepository struct {
	refreshColl      *mongo.Collection
	verificationColl *mongo.Collection
}

// NewTokenRepository creates a MongoDB-backed TokenRepository.
func NewTokenRepository(db *mongo.Database) domain.TokenRepository {
	return &tokenRepository{
		refreshColl:      db.Collection("refresh_tokens"),
		verificationColl: db.Collection("verification_tokens"),
	}
}

// --- Refresh Tokens ---

func (r *tokenRepository) StoreRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.refreshColl.InsertOne(ctx, token)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to store refresh token", 500, err)
	}
	return nil
}

func (r *tokenRepository) FindRefreshToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	filter := bson.M{"token": token, "revoked": false}
	if err := r.refreshColl.FindOne(ctx, filter).Decode(&rt); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrUnauthorized
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find refresh token", 500, err)
	}
	return &rt, nil
}

func (r *tokenRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	filter := bson.M{"token": token}
	update := bson.M{"$set": bson.M{"revoked": true}}
	_, err := r.refreshColl.UpdateMany(ctx, filter, update)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to revoke refresh token", 500, err)
	}
	return nil
}

func (r *tokenRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	filter := bson.M{"user_id": userID, "revoked": false}
	update := bson.M{"$set": bson.M{"revoked": true}}
	_, err := r.refreshColl.UpdateMany(ctx, filter, update)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to revoke user refresh tokens", 500, err)
	}
	return nil
}

// --- Verification Tokens ---

func (r *tokenRepository) StoreVerificationToken(ctx context.Context, token *domain.VerificationToken) error {
	_, err := r.verificationColl.InsertOne(ctx, token)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to store verification token", 500, err)
	}
	return nil
}

func (r *tokenRepository) FindVerificationToken(ctx context.Context, token string, tokenType domain.TokenType) (*domain.VerificationToken, error) {
	var vt domain.VerificationToken
	filter := bson.M{"token": token, "type": tokenType, "used": false}
	if err := r.verificationColl.FindOne(ctx, filter).Decode(&vt); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find verification token", 500, err)
	}
	return &vt, nil
}

func (r *tokenRepository) MarkVerificationTokenUsed(ctx context.Context, id string) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"used": true}}
	_, err := r.verificationColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to mark verification token used", 500, err)
	}
	return nil
}

// EnsureTokenIndexes creates required indexes on token collections.
func EnsureTokenIndexes(ctx context.Context, db *mongo.Database) error {
	refreshColl := db.Collection("refresh_tokens")
	refreshIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_refresh_token_unique"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_refresh_user_id"),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("idx_refresh_expires"),
		},
	}
	_, err := refreshColl.Indexes().CreateMany(ctx, refreshIndexes)
	if err != nil {
		return fmt.Errorf("failed to create refresh token indexes: %w", err)
	}

	verificationColl := db.Collection("verification_tokens")
	verificationIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}, {Key: "type", Value: 1}},
			Options: options.Index().SetName("idx_verify_token_type"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_verify_user_id"),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("idx_verify_expires"),
		},
	}
	_, err = verificationColl.Indexes().CreateMany(ctx, verificationIndexes)
	if err != nil {
		return fmt.Errorf("failed to create verification token indexes: %w", err)
	}

	return nil
}
