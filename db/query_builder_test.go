package db

import (
	"testing"

	"github.com/lib/pq"
	"github.com/rm-hull/gps-routes-api/models/common"
	"github.com/rm-hull/gps-routes-api/models/request"
	"github.com/stretchr/testify/assert"
)

type Result struct {
	sql    string
	params []interface{}
}

func TestQueryBuilder_Build(t *testing.T) {

	tests := []struct {
		name string
		qb   *QueryBuilder
		want Result
	}{
		{
			name: "empty query",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}),
			want: Result{
				sql:    "SELECT * FROM routes",
				params: []interface{}{},
			},
		},
		{
			name: "single word query",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{Query: "test"}),
			want: Result{
				sql:    "SELECT * FROM routes WHERE search_vector @@ to_tsquery($1)",
				params: []interface{}{"test:*"},
			},
		},
		{
			name: "multiple word query",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{Query: "hello world"}),
			want: Result{
				sql:    "SELECT * FROM routes WHERE search_vector @@ to_tsquery($1)",
				params: []interface{}{"hello:* & world:*"},
			},
		},
		{
			name: "bounding box",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{BoundingBox: []float64{1.9, 2.8, 3.7, 4.6}}),
			want: Result{
				sql:    "SELECT * FROM routes WHERE ST_Within(_geoloc, ST_MakeEnvelope($1, $2, $3, $4, 4326))",
				params: []interface{}{1.9, 2.8, 3.7, 4.6},
			},
		},
		{
			name: "nearest",
			qb: NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{
				Nearby: &request.Nearby{
					Center: &common.GeoLoc{
						Latitude:  1.234,
						Longitude: 5.678,
					},
					DistanceKm: 20,
				},
			}),
			want: Result{
				sql:    "SELECT * FROM routes WHERE ST_DWithin(ST_Transform(_geoloc, 3857), ST_Transform(ST_SetSRID(ST_Point($1, $2), 4326), 3857), $3)",
				params: []interface{}{5.678, 1.234, int32(20000)},
			},
		},
		{
			name: "with where clause",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).WithWhereClause("name = 'test'"),
			want: Result{
				sql:    "SELECT * FROM routes WHERE name = 'test'",
				params: []interface{}{},
			},
		},
		{
			name: "with multiple where clauses",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).WithWhereClause("name = 'test'").WithWhereClause("type = 'hike'"),
			want: Result{
				sql:    "SELECT * FROM routes WHERE name = 'test' AND type = 'hike'",
				params: []interface{}{},
			},
		},
		{
			name: "with group by",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).WithGroupBy("name"),
			want: Result{
				sql:    "SELECT * FROM routes GROUP BY name",
				params: []interface{}{},
			},
		},
		{
			name: "with order by",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).WithOrderBy("name DESC"),
			want: Result{
				sql:    "SELECT * FROM routes ORDER BY name DESC",
				params: []interface{}{},
			},
		},
		{
			name: "with limit",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).WithLimit(10),
			want: Result{
				sql:    "SELECT * FROM routes LIMIT $1",
				params: []interface{}{int32(10)},
			},
		},
		{
			name: "with no limit",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).WithLimit(-1),
			want: Result{
				sql:    "SELECT * FROM routes",
				params: []interface{}{},
			},
		},
		{
			name: "with offset",
			qb:   NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).WithOffset(20),
			want: Result{
				sql:    "SELECT * FROM routes OFFSET $1",
				params: []interface{}{int32(20)},
			},
		},
		{
			name: "with all",
			qb: NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{}).
				WithWhereClause("name = 'test'").
				WithGroupBy("name").
				WithOrderBy("name DESC").
				WithLimit(10).
				WithOffset(20),
			want: Result{
				sql:    "SELECT * FROM routes WHERE name = 'test' GROUP BY name ORDER BY name DESC OFFSET $2 LIMIT $1",
				params: []interface{}{int32(10), int32(20)},
			},
		},
		{
			name: "with facets",
			qb: NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{
				Facets: map[string][]string{
					"type": {"hike", "bike"},
				},
			}),
			want: Result{
				sql:    "SELECT * FROM routes WHERE type = ANY($1)",
				params: []interface{}{pq.Array([]string{"hike", "bike"})},
			},
		},
		{
			name: "with excluded facets",
			qb: NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{
				Facets: map[string][]string{
					"type": {"hike", "bike"},
					"name": {"test"},
				},
			}).WithExcludeFacets("type"),
			want: Result{
				sql:    "SELECT * FROM routes WHERE name = ANY($1)",
				params: []interface{}{pq.Array([]string{"test"})},
			},
		},
		{
			name: "with array fields",
			qb: NewQueryBuilder("SELECT * FROM routes", &request.SearchRequest{
				Facets: map[string][]string{
					"type": {"hike", "bike"},
				},
			}).WithArrayFields("type"),
			want: Result{
				sql:    "SELECT * FROM routes WHERE type && $1::TEXT[]",
				params: []interface{}{pq.Array([]string{"hike", "bike"})},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params := tt.qb.Build()

			assert.Equal(t, tt.want.sql, sql)
			assert.Equal(t, tt.want.params, params)
		})
	}
}
