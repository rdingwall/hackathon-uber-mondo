package main

import (
	"bytes"
	"text/template"
)

type coordinate struct {
	Latitude  float64
	Longitude float64
}

var googleMapsImageUrlTemplate, _ = template.New("googleMapsImageUrl").Parse("https://maps.googleapis.com/maps/api/staticmap?style=feature:landscape|visibility:off&style=feature:poi|visibility:off&style=feature:transit|visibility:off&style=feature:road.highway|element:geometry|lightness:39&style=feature:road.local|element:geometry|gamma:1.45&style=feature:road|element:labels|gamma:1.22&style=feature:administrative|visibility:off&style=feature:administrative.locality|visibility:on&style=feature:landscape.natural|visibility:on&scale=2&markers=shadow:false|scale:2|icon:http://d1a3f4spazzrp4.cloudfront.net/receipt-new/marker-start@2x.png|{{.Start.Latitude}},{{.Start.Longitude}}&markers=shadow:false|scale:2|icon:http://d1a3f4spazzrp4.cloudfront.net/receipt-new/marker-finish@2x.png|{{.End.Latitude}},{{.End.Longitude}}&path=color:0x2dbae4ff|weight:4|{{.Start.Latitude}},{{.Start.Longitude}}|{{.End.Latitude}},{{.End.Longitude}}&size=400x400&key={{.ApiKey}}&zoom=12")

func googleMapsUrl(start, end coordinate, apiKey string) string {
	data := struct {
		Start  coordinate
		End    coordinate
		ApiKey string
	}{Start: start, End: end, ApiKey: apiKey}
	var buffer bytes.Buffer
	err := googleMapsImageUrlTemplate.Execute(&buffer, data)
	if err != nil {
		panic(err)
	}
	return buffer.String()
}
