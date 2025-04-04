package db

import (
	"reflect"
	"testing"

	"github.com/lib/pq"
	model "github.com/rm-hull/gps-routes-api/go"
)

func TestQueryBuilder_Build(t *testing.T) {

	tests := []struct {
		name  string
		qb    *QueryBuilder
		want  string
		want1 []interface{}
	}{
		{
			name:  "empty query",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}),
			want:  "SELECT * FROM routes",
			want1: []interface{}{},
		},
		{
			name:  "single word query",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{Query: "test"}),
			want:  "SELECT * FROM routes WHERE search_vector @@ to_tsquery($1)",
			want1: []interface{}{"test:*"},
		},
		{
			name:  "multiple word query",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{Query: "hello world"}),
			want:  "SELECT * FROM routes WHERE search_vector @@ to_tsquery($1)",
			want1: []interface{}{"hello:* & world:*"},
		},
		{
			name:  "bounding box",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{BoundingBox: []float64{1.9, 2.8, 3.7, 4.6}}),
			want:  "SELECT * FROM routes WHERE ST_Within(_geoloc, ST_MakeEnvelope($1, $2, $3, $4, 4326))",
			want1: []interface{}{1.9, 2.8, 3.7, 4.6},
		},
		{
			name:  "with where clause",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}).WithWhereClause("name = 'test'"),
			want:  "SELECT * FROM routes WHERE name = 'test'",
			want1: []interface{}{},
		},
		{
			name:  "with multiple where clauses",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}).WithWhereClause("name = 'test'").WithWhereClause("type = 'hike'"),
			want:  "SELECT * FROM routes WHERE name = 'test' AND type = 'hike'",
			want1: []interface{}{},
		},
		{
			name:  "with group by",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}).WithGroupBy("name"),
			want:  "SELECT * FROM routes GROUP BY name",
			want1: []interface{}{},
		},
		{
			name:  "with order by",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}).WithOrderBy("name"),
			want:  "SELECT * FROM routes ORDER BY name DESC",
			want1: []interface{}{},
		},
		{
			name:  "with limit",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}).WithLimit(10),
			want:  "SELECT * FROM routes LIMIT $1",
			want1: []interface{}{int32(10)},
		},
		{
			name:  "with offset",
			qb:    NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}).WithOffset(20),
			want:  "SELECT * FROM routes OFFSET $1",
			want1: []interface{}{int32(20)},
		},
		{
			name: "with all",
			qb: NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{}).
				WithWhereClause("name = 'test'").
				WithGroupBy("name").
				WithOrderBy("name").
				WithLimit(10).
				WithOffset(20),
			want:  "SELECT * FROM routes WHERE name = 'test' GROUP BY name ORDER BY name DESC OFFSET $2 LIMIT $1",
			want1: []interface{}{int32(10), int32(20)},
		},
		{
			name: "with facets",
			qb: NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{
				Facets: map[string][]string{
					"type": {"hike", "bike"},
				},
			}),
			want:  "SELECT * FROM routes WHERE type = ANY($1)",
			want1: []interface{}{pq.Array([]string{"hike", "bike"})},
		},
		{
			name: "with excluded facets",
			qb: NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{
				Facets: map[string][]string{
					"type": {"hike", "bike"},
					"name": {"test"},
				},
			}).WithExcludeFacets("type"),
			want:  "SELECT * FROM routes WHERE name = ANY($1)",
			want1: []interface{}{pq.Array([]string{"test"})},
		},
		{
			name: "with array fields",
			qb: NewQueryBuilder("SELECT * FROM routes", &model.SearchRequest{
				Facets: map[string][]string{
					"type": {"hike", "bike"},
				},
			}).WithArrayFields("type"),
			want:  "SELECT * FROM routes WHERE type && $1::TEXT[]",
			want1: []interface{}{pq.Array([]string{"hike", "bike"})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, got1 := tt.qb.Build()
			if got != tt.want {
				t.Errorf("QueryBuilder.Build() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("QueryBuilder.Build() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
