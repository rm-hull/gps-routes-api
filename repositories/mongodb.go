package repositories

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	model "github.com/rm-hull/gps-routes-api/go"
)

type MongoDbRepository struct {
	collection *mongo.Collection
}

func NewMongoRouteRepository(client *mongo.Client, dbName, collectionName string) *MongoDbRepository {
	collection := client.Database(dbName).Collection(collectionName)
	return &MongoDbRepository{collection: collection}
}

func (repo *MongoDbRepository) Store(ctx context.Context, route *model.RouteMetadata) error {
	panic("unimplemented")
}

func (repo *MongoDbRepository) FindByObjectID(ctx context.Context, objectID string) (*model.RouteMetadata, error) {
	var routeMetadata model.RouteMetadata
	filter := bson.M{"objectID": objectID}
	err := repo.collection.FindOne(ctx, filter).Decode(&routeMetadata)
	if err == mongo.ErrNoDocuments {
		return nil, nil // No document found
	}
	if err != nil {
		return nil, err
	}
	return &routeMetadata, nil
}

func buildQuery(criteria *model.SearchRequest) bson.M {
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

	switch len(filters) {
	case 0:
		return bson.M{}
	case 1:
		return filters[0]
	default:
		return bson.M{"$and": filters}
	}
}

func sortOrder(criteria *model.SearchRequest) bson.M {
	if criteria.Query == "" {
		return bson.M{"created_at": -1}
	}

	return bson.M{"score": bson.M{"$meta": "textScore"}}
}

func (repo *MongoDbRepository) SearchHits(ctx context.Context, criteria *model.SearchRequest) (*[]model.RouteSummary, error) {
	options := options.Find().
		SetSort(sortOrder(criteria)).
		SetLimit(int64(criteria.Limit)).
		SetSkip(int64(criteria.Offset)).
		SetProjection(bson.M{
			"objectID":           1,
			"ref":                1,
			"title":              1,
			"description":        1,
			"headline_image_url": 1,
			"location":           1,
		})

	cursor, err := repo.collection.Find(ctx, buildQuery(criteria), options)
	if err != nil {
		return nil, fmt.Errorf("failed while finding with search query: %v", err)
	}
	defer cursor.Close(ctx)

	var results []model.RouteSummary
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed while marshalling results: %v", err)
	}
	return &results, nil

}

func (repo *MongoDbRepository) CountAll(ctx context.Context, criteria *model.SearchRequest) (int64, error) {
	total, err := repo.collection.CountDocuments(ctx, buildQuery(criteria))
	if err != nil {
		return 0, fmt.Errorf("failed while counting matching documents: %v", err)
	}
	return total, nil
}

func (repo *MongoDbRepository) FacetCounts(ctx context.Context, criteria *model.SearchRequest, facetField string, limit int) (*map[string]int64, error) {
	pipeline := []bson.M{
		{
			"$match": buildQuery(criteria),
		},
		{
			"$group": bson.M{
				"_id":   "$" + facetField,
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$sort": bson.M{"count": -1},
		},
		{
			"$limit": limit,
		},
		{
			"$match": bson.M{"_id": bson.M{"$ne": nil}},
		},
	}

	cursor, err := repo.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %v", err)
	}
	defer cursor.Close(ctx)

	type resultType struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}

	var results []resultType
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %v", err)
	}

	counts := make(map[string]int64, 0)
	for _, result := range results {
		counts[result.ID] = result.Count
	}

	return &counts, nil
}
