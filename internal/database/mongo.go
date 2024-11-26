package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	client   *mongo.Client
	database string
}

func NewMongoDB(cfg *config.Config) (*MongoDB, error) {
	uri := cfg.MongoDB.URI
	uri = strings.Replace(uri, "<db_password>", cfg.MongoDB.Password, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	return &MongoDB{
		client:   client,
		database: cfg.MongoDB.Database,
	}, nil
}

func (db *MongoDB) UpsertGuild(guild *models.Guild) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.client.Database(db.database).Collection("guilds")

	filter := bson.M{"guild_id": guild.GuildID}
	update := bson.M{"$set": guild}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (db *MongoDB) UpdateGuildStatus(guildID string, isActive bool, leftAt *time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.client.Database(db.database).Collection("guilds")

	update := bson.M{
		"$set": bson.M{
			"is_active":    isActive,
			"left_at":      leftAt,
			"last_updated": time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"guild_id": guildID}, update)
	return err
}

func (db *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.client.Disconnect(ctx)
}

func (db *MongoDB) UpdateMemberCount(guildID string, delta int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.client.Database(db.database).Collection("guilds")

	update := bson.M{
		"$inc": bson.M{
			"member_count": delta,
		},
		"$set": bson.M{
			"last_updated": time.Now(),
		},
	}

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"guild_id": guildID},
		update,
	)
	return err
}

func (db *MongoDB) UpdateGuildSettings(guildID string, setting string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.client.Database(db.database).Collection("guilds")

	update := bson.M{
		"$set": bson.M{
			fmt.Sprintf("settings.%s", setting): value,
			"last_updated":                      time.Now(),
		},
	}

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"guild_id": guildID},
		update,
	)
	return err
}
