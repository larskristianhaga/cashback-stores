package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("App live and listening on port:", port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Request received")

		log.Println("Fetching SAS shops")
		getSasShops := getSasShops()
		log.Println("Found", len(getSasShops.Data), "SAS shops")

		log.Println("Fetching ViaTrumf shops")
		getViatrumfShops := getViatrumfShops()
		log.Println("Found", len(getViatrumfShops.Data), "ViaTrumf shops")

		// Combine the two data sets into one
		combinedData := struct {
			SASShopsData     SASShopsData
			ViaTrumfShopData ViaTrumfShopData
		}{
			SASShopsData:     SASShopsData{},
			ViaTrumfShopData: ViaTrumfShopData{},
		}

		combinedData.SASShopsData = getSasShops
		combinedData.ViaTrumfShopData = getViatrumfShops

		// Convert the combined data to JSON
		combinedDataJSON, err := json.Marshal(combinedData)
		if err != nil {
			log.Fatal(err.Error())
		}

		// Write the JSON to the response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(combinedDataJSON)
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getSasShops() SASShopsData {
	var shopUrl = "https://onlineshopping.loyaltykey.com/api/v1/shops?filter[channel]=SAS&filter[language]=nb&filter[country]=NO&filter[amount]=5000&filter[compressed]=true"

	response, err := http.Get(shopUrl)
	if err != nil {
		log.Fatal(err.Error())
	}

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	var sasShops SASShopsData
	err = json.Unmarshal(responseData, &sasShops)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Add source
	for i := range sasShops.Data {
		sasShops.Data[i].Source = "sasonlineshopping"
	}

	return sasShops
}

func getViatrumfShops() ViaTrumfShopData {
	var shopUrl = "https://trumfnetthandel.no/category/paged/all/999/0/popularity/"

	response, err := http.Get(shopUrl)
	if err != nil {
		log.Fatal(err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Extract elements
	var elements []string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		dataName, exists := s.Attr("data-name")
		if exists {
			elements = append(elements, dataName)
		}
	})

	// Return extracted elements, as a JSON object, and add source
	var shops []ViaTrumfShop
	for _, element := range elements {
		shops = append(shops, ViaTrumfShop{Name: element, Source: "trumfnetthandel"})
	}

	return ViaTrumfShopData{Data: shops}
}

type SASShopsData struct {
	Data []SASShop `json:"data"`
}

type SASShop struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Source string
}

type ViaTrumfShopData struct {
	Data []ViaTrumfShop `json:"data"`
}

type ViaTrumfShop struct {
	Name   string `json:"name"`
	Source string
}

type APIResponse struct {
	Data []APIResponseData `json:"data"`
}

type APIResponseData struct {
	Name   string   `json:"name"`
	Source []string `json:"source"`
	SASShopExtra
	ViaTrumfShopExtra
}

type SASShopExtra struct {
	UUID string `json:"uuid"`
	Slug string `json:"slug"`
}

type ViaTrumfShopExtra struct {
	Slug string `json:"slug"`
}
