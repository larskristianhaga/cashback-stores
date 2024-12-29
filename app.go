package main

import (
	"crypto/tls"
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

	http.HandleFunc("/", RootHandler)
	http.HandleFunc("/ping", PingHandler)
	http.HandleFunc("/health", HealthHandler)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func RootHandler(w http.ResponseWriter, _ *http.Request) {
	log.Println("Request received")

	log.Println("Fetching SAS shops")
	getSasShops := getSasShops()
	log.Println("Found", len(getSasShops.Data), "SAS shops")

	log.Println("Fetching ViaTrumf shops")
	getViatrumfShops := getViatrumfShops()
	log.Println("Found", len(getViatrumfShops.Data), "ViaTrumf shops")

	// Combine the data into a single JSON object of type APIResponse.
	var combinedDataMap = make(map[string]APIResponseData)

	for _, sasShop := range getSasShops.Data {
		combinedDataMap[sasShop.Name] = APIResponseData{
			Name:   sasShop.Name,
			Source: []string{sasShop.Source},
			SasOnlineShoppingExtra: SasOnlineShoppingExtra{
				UUID: sasShop.UUID,
				Slug: sasShop.Slug,
			},
		}
	}

	for _, viaTrumfShop := range getViatrumfShops.Data {
		if data, exists := combinedDataMap[viaTrumfShop.Name]; exists {
			data.Source = append(data.Source, viaTrumfShop.Source)
			data.TrumfNetthandelExtra = TrumfNetthandelExtra{
				Slug: viaTrumfShop.Name,
			}
			combinedDataMap[viaTrumfShop.Name] = data
		} else {
			combinedDataMap[viaTrumfShop.Name] = APIResponseData{
				Name:   viaTrumfShop.Name,
				Source: []string{viaTrumfShop.Source},
				TrumfNetthandelExtra: TrumfNetthandelExtra{
					Slug: viaTrumfShop.Name,
				},
			}
		}
	}

	var combinedData []APIResponseData
	for _, data := range combinedDataMap {
		combinedData = append(combinedData, data)
	}

	combinedDataJSON, err := json.Marshal(APIResponse{Data: combinedData})
	if err != nil {
		log.Fatal(err.Error())
	}

	// Write the JSON to the response
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(combinedDataJSON)
}

func PingHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("pong"))
}

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("I'm healthy"))
}

func getSasShops() SASShopsData {
	var shopUrl = "https://onlineshopping.loyaltykey.com/api/v1/shops?filter[channel]=SAS&filter[language]=nb&filter[country]=NO&filter[amount]=5000&filter[compressed]=true"

	client := createInsecureHTTPClient()

	response, err := client.Get(shopUrl)
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
		sasShops.Data[i].Source = Sasonlineshopping
	}

	return sasShops
}

func getViatrumfShops() ViaTrumfShopData {
	var shopUrl = "https://trumfnetthandel.no/category/paged/all/999/0/popularity/"

	client := createInsecureHTTPClient()

	response, err := client.Get(shopUrl)
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
		shops = append(shops, ViaTrumfShop{Name: element, Source: Trumfnetthandel})
	}

	return ViaTrumfShopData{Data: shops}
}

func createInsecureHTTPClient() *http.Client {
	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: customTransport}
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
	Name                   string                 `json:"name"`
	Source                 []string               `json:"source"`
	TrumfNetthandelExtra   TrumfNetthandelExtra   `json:"trumfnetthandel_extra"`
	SasOnlineShoppingExtra SasOnlineShoppingExtra `json:"sasonlineshopping_extra"`
}

type TrumfNetthandelExtra struct {
	Slug string `json:"slug"`
}

type SasOnlineShoppingExtra struct {
	UUID string `json:"uuid"`
	Slug string `json:"slug"`
}

const (
	Trumfnetthandel   = "trumfnetthandel"
	Sasonlineshopping = "sasonlineshopping"
)
