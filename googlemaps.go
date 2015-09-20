package main

import (
	"bytes"
	"text/template"
)

type coordinate struct {
	Latitude  float64
	Longitude float64
}

var googleMapsImageUrlTemplate, _ = template.New("googleMapsImageUrl").Parse("https://maps-api-ssl.google.com/maps/api/staticmap?style=feature%3Alandscape%7Cvisibility%3Aoff&style=feature%3Apoi%7Cvisibility%3Aoff&style=feature%3Atransit%7Cvisibility%3Aoff&style=feature%3Aroad.highway%7Celement%3Ageometry%7Clightness%3A39&style=feature%3Aroad.local%7Celement%3Ageometry%7Cgamma%3A1.45&style=feature%3Aroad%7Celement%3Alabels%7Cgamma%3A1.22&style=feature%3Aadministrative%7Cvisibility%3Aoff&style=feature%3Aadministrative.locality%7Cvisibility%3Aon&style=feature%3Alandscape.natural%7Cvisibility%3Aon&scale=2&markers=shadow%3Afalse%7Cscale%3A2%7Cicon%3Ahttp%3A%2F%2Fd1a3f4spazzrp4.cloudfront.net%2Freceipt-new%2Fmarker-start%402x.png%7C{{.Start.Latitude}},%2C{{.Start.Longitude}}&markers=shadow%3Afalse%7Cscale%3A2%7Cicon%3Ahttp%3A%2F%2Fd1a3f4spazzrp4.cloudfront.net%2Freceipt-new%2Fmarker-finish%402x.png%7C{{.End.Latitude}}%2C{{.End.Longitude}}&path=color%3A0x2dbae4ff%7Cweight%3A4%7C{{.Start.Latitude}}%2C{{.Start.Longitude}}%7C{{.End.Latitude}}%2C{{.End.Longitude}}&size=400x400&key={{.ApiKey}}&zoom=12")

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
