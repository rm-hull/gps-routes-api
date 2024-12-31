package repositories

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	openapi "github.com/rm-hull/gps-routes-api/go"
)

type RouteRepository interface {
	FindByObjectID(ctx context.Context, objectID string) (*openapi.RouteMetadata, error)
}

type MongoRouteRepository struct {
	collection *mongo.Collection
}

func NewMongoRouteRepository(client *mongo.Client, dbName, collectionName string) *MongoRouteRepository {
	collection := client.Database(dbName).Collection(collectionName)
	return &MongoRouteRepository{collection: collection}
}

func (r *MongoRouteRepository) FindByObjectID(ctx context.Context, objectID string) (*openapi.RouteMetadata, error) {
	var routeMetadata openapi.RouteMetadata
	filter := bson.M{"objectID": objectID}
	err := r.collection.FindOne(ctx, filter).Decode(&routeMetadata)
	if err == mongo.ErrNoDocuments {
		return nil, nil // No document found
	}
	if err != nil {
		return nil, err
	}
	return &routeMetadata, nil
}
