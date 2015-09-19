package main

import (
	"bytes"
	"text/template"
)

type coordinate struct {
	latitude  float64
	longitude float64
}

var googleMapsImageUrlTemplate, _ = template.New("googleMapsImageUrl").Parse("https://maps.googleapis.com/maps/api/staticmap?style=feature:landscape|visibility:off&style=feature:poi|visibility:off&style=feature:transit|visibility:off&style=feature:road.highway|element:geometry|lightness:39&style=feature:road.local|element:geometry|gamma:1.45&style=feature:road|element:labels|gamma:1.22&style=feature:administrative|visibility:off&style=feature:administrative.locality|visibility:on&style=feature:landscape.natural|visibility:on&scale=2&markers=shadow:false|scale:2|icon:http://d1a3f4spazzrp4.cloudfront.net/receipt-new/marker-start@2x.png|{{.start.latitude}},{{start.longitude}}&markers=shadow:false|scale:2|icon:http://d1a3f4spazzrp4.cloudfront.net/receipt-new/marker-finish@2x.png|{{end.latitude}},{{end.longitude}}&path=color:0x2dbae4ff|weight:4|{{.start.latitude}},{{start.longitude}}|{{end.latitude}},{{end.longitude}}&size=400x400&key={{apiKey}}&zoom=12")

func googleMapsUrl(start, end coordinate, apiKey string) string {
	data := struct {
		start  coordinate
		end    coordinate
		apiKey string
	}{start: start, end: end, apiKey: apiKey}
	var buffer bytes.Buffer
	err := googleMapsImageUrlTemplate.Execute(&buffer, data)
	if err != nil {
		panic(err)
	}
	return buffer.String()
}
