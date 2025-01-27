package db

import (
	"fmt"
	"strings"

	model "github.com/rm-hull/gps-routes-api/go"
)

type QueryBuilder struct {
	criteria       *model.SearchRequest
	selectPart     string
	whereClauses   []string
	excludedFacets []string
	params         []interface{}
	orderBy        string
	groupBy        string
	limit          string
	offset         string
}

func NewQueryBuilder(selectPart string, criteria *model.SearchRequest) *QueryBuilder {
	qb := &QueryBuilder{
		selectPart: selectPart,
		criteria:   criteria,
		params:     make([]interface{}, 0),
	}

	return qb.applyWhereConditions()
}

func (qb *QueryBuilder) WithWhereClause(whereClause string) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, whereClause)
	return qb
}

func (qb *QueryBuilder) WithOrderBy(fieldName string) *QueryBuilder {
	if qb.orderBy != "" {
		panic("unexpected: orderBy value already set")
	}

	qb.orderBy = fmt.Sprintf("ORDER BY %s DESC", fieldName)
	return qb
}

func (qb *QueryBuilder) WithGroupBy(fieldName string) *QueryBuilder {
	if qb.groupBy != "" {
		panic("unexpected: groupBy value already set")
	}

	qb.groupBy = fmt.Sprintf("GROUP BY %s", fieldName)
	return qb
}

func (qb *QueryBuilder) WithOffset(offset int32) *QueryBuilder {
	if qb.offset != "" {
		panic("unexpected: offset value already set")
	}

	placeholder := len(qb.params) + 1
	qb.offset = fmt.Sprintf("OFFSET $%d", placeholder)
	qb.params = append(qb.params, offset)
	return qb
}

func (qb *QueryBuilder) WithLimit(limit int32) *QueryBuilder {
	if qb.limit != "" {
		panic("unexpected: limit value already set")
	}

	placeholder := len(qb.params) + 1
	qb.limit = fmt.Sprintf("LIMIT $%d", placeholder)
	qb.params = append(qb.params, limit)
	return qb
}

func (qb *QueryBuilder) WithExcludeFacets(facetNames ...string) *QueryBuilder {
	qb.excludedFacets = facetNames
	return qb
}

func (qb *QueryBuilder) Build() (string, []interface{}) {

	if qb.criteria.Facets != nil {
		for facet, values := range qb.criteria.Facets {
			excluded := false

			for _, name := range qb.excludedFacets {
				if name == facet {
					excluded = true
					break
				}
			}

			if !excluded {
				qb.WithWhereClause(fmt.Sprintf("%s = ANY($%d)", facet, len(qb.params)+1))
				qb.params = append(qb.params, values)
			}
		}
	}

	var whereClause string
	if len(qb.whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(qb.whereClauses, " AND ")
	}

	return fmt.Sprintf("%s %s %s %s %s %s", qb.selectPart, whereClause, qb.groupBy, qb.orderBy, qb.offset, qb.limit), qb.params
}

// split the query into words and suffix each word with ':*' to allow prefix matching, then join them with '&'
func prefix(query string) string {
	words := strings.Split(query, " ")
	for i, word := range words {
		words[i] = word + ":*"
	}
	return strings.Join(words, " & ")
}

func (qb *QueryBuilder) applyWhereConditions() *QueryBuilder {

	if qb.criteria.Query != "" {
		qb.WithWhereClause("search_vector @@ to_tsquery($1)")
		qb.params = append(qb.params, prefix(qb.criteria.Query))
	}

	if qb.criteria.BoundingBox != nil {
		offsetPlaceholder := len(qb.params) + 1
		qb.WithWhereClause(fmt.Sprintf("ST_Within(_geoloc, ST_MakeEnvelope($%d, $%d, $%d, $%d, 4326))", offsetPlaceholder, offsetPlaceholder+1, offsetPlaceholder+2, offsetPlaceholder+3))
		for _, value := range qb.criteria.BoundingBox {
			qb.params = append(qb.params, value)
		}
	}

	return qb
}
