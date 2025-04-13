package osdatahub

import (
	model "github.com/rm-hull/gps-routes-api/go"
	"github.com/wroge/wgs84"
)

var epsg = wgs84.EPSG().Code(4326)
var bng = wgs84.OSGB36NationalGrid()

func ToWSG84(result *Result) *model.GeoLoc {
	if result == nil {
		return nil
	}

	lng, lat, _ := bng.To(epsg).Round(10)(
		result.GazetteerEntry.GeometryX,
		result.GazetteerEntry.GeometryY,
		0,
	)

	return &model.GeoLoc{Latitude: lat, Longitude: lng}
}
