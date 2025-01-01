package repositories

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	openapi "github.com/rm-hull/gps-routes-api/go"
)

type RouteRepository interface {
	FindByObjectID(ctx context.Context, objectID string) (*openapi.RouteMetadata, error)
	CountAll(ctx context.Context, criteria *openapi.SearchRequest) (int64, error)
	SearchHits(ctx context.Context, criteria *openapi.SearchRequest) (*[]openapi.RouteSummary, error)
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

func buildQuery(criteria *openapi.SearchRequest) bson.M {
	filters := make([]bson.M, 0)

	if criteria.Query != "" {
		filters = append(filters, bson.M{
			"$text": bson.M{
				"$search": criteria.Query,
			},
		})
	}

	if criteria.BoundingBox != nil {
		filters = append(filters, bson.M{
			"location": bson.M{
				"$geoWithin": bson.M{
					"$box": [][]float64{
						{criteria.BoundingBox[0], criteria.BoundingBox[1]},
						{criteria.BoundingBox[2], criteria.BoundingBox[3]},
					},
				},
			},
		})
	}

	if criteria.Facets != nil {
		for facet, values := range criteria.Facets {
			filters = append(filters, bson.M{
				facet: bson.M{
					"$in": values,
				},
			})
		}
	}

	if len(filters) > 0 {
		return bson.M{"$and": filters}
	}

	return bson.M{}
}

func sortOrder(criteria *openapi.SearchRequest) bson.M {
	if criteria.Query == "" {
		return bson.M{"created_at": -1}
	}

	return bson.M{"score": bson.M{"$meta": "textScore"}}
}

func (r *MongoRouteRepository) SearchHits(ctx context.Context, criteria *openapi.SearchRequest) (*[]openapi.RouteSummary, error) {
	options := options.Find().
		SetSort(sortOrder(criteria)).
		SetLimit(int64(criteria.Limit)).
		SetSkip(int64(criteria.Offset)).
		SetProjection(bson.M{
			"object_id":          1,
			"ref":                1,
			"title":              1,
			"description":        1,
			"headline_image_url": 1,
			"location":           1,
		})

	cursor, err := r.collection.Find(ctx, buildQuery(criteria), options)
	if err != nil {
		return nil, fmt.Errorf("failed while finding with search query: %v", err)
	}
	defer cursor.Close(ctx)

	var results []openapi.RouteSummary
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed while marshalling results: %v", err)
	}
	return &results, nil

}

func (r *MongoRouteRepository) CountAll(ctx context.Context, criteria *openapi.SearchRequest) (int64, error) {
	total, err := r.collection.CountDocuments(ctx, buildQuery(criteria))
	if err != nil {
		return 0, fmt.Errorf("failed while counting matching documents: %v", err)
	}
	return total, nil
}
