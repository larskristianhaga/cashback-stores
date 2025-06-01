package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("App live and listening on port:", port)

	http.HandleFunc("/", loggingMiddleware(RootHandler))
	http.HandleFunc("/ping", loggingMiddleware(PingHandler))
	http.HandleFunc("/health", loggingMiddleware(HealthHandler))

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

	log.Println("Combining data")
	combinedUniqueData := combineData(getSasShops, getViatrumfShops)

	log.Println("Marshalling data")
	combinedDataJSON, err := json.Marshal(APIResponse{Data: combinedUniqueData})
	if err != nil {
		log.Fatal(err.Error())
	}

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

func combineData(sasShops SASShopsData, viaTrumfShops ViaTrumfShopData) []APIResponseData {
	sasStoreBaseURL := "https://onlineshopping.flysas.com/nb-NO/butikker"
	trumfnetthandelBaseUrl := "https://trumfnetthandel.no/cashback"

	var combinedDataMap = make(map[string]APIResponseData)

	for _, sasShop := range sasShops.Data {

		combinedDataMap[sasShop.Name] = APIResponseData{
			Name:   sasShop.Name,
			Source: []string{sasShop.Source},
		}
		if sasShop.Source == Sasonlineshopping {
			data := combinedDataMap[sasShop.Name]
			data.SasOnlineShoppingExtra = &SasOnlineShoppingExtra{
				UUID: sasShop.UUID,
				Slug: sasShop.Slug,
				Url:  fmt.Sprintf("%s/%s/%s", sasStoreBaseURL, sasShop.Slug, sasShop.UUID),
			}
			combinedDataMap[sasShop.Name] = data
		}

	}

	for _, viaTrumfShop := range viaTrumfShops.Data {

		if data, exists := combinedDataMap[viaTrumfShop.Name]; exists {
			data.Source = append(data.Source, viaTrumfShop.Source)
			if viaTrumfShop.Source == Trumfnetthandel {
				data.TrumfNetthandelExtra = &TrumfNetthandelExtra{
					Slug: viaTrumfShop.Name,
					Url:  fmt.Sprintf("%s/%s", trumfnetthandelBaseUrl, viaTrumfShop.Name),
				}
			}
			combinedDataMap[viaTrumfShop.Name] = data
		} else {
			combinedDataMap[viaTrumfShop.Name] = APIResponseData{
				Name:   viaTrumfShop.Name,
				Source: []string{viaTrumfShop.Source},
			}
			if viaTrumfShop.Source == Trumfnetthandel {
				data := combinedDataMap[viaTrumfShop.Name]
				data.TrumfNetthandelExtra = &TrumfNetthandelExtra{
					Slug: viaTrumfShop.Name,
					Url:  fmt.Sprintf("%s/%s", trumfnetthandelBaseUrl, viaTrumfShop.Name),
				}
				combinedDataMap[viaTrumfShop.Name] = data
			}
		}

	}

	var combinedData []APIResponseData
	for _, data := range combinedDataMap {
		if len(data.Source) > 0 {
			if !contains(data.Source, Trumfnetthandel) {
				data.TrumfNetthandelExtra = &TrumfNetthandelExtra{}
			}
			combinedData = append(combinedData, data)
		}
	}

	return combinedData
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
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
	Name                   string                  `json:"name"`
	Source                 []string                `json:"source"`
	TrumfNetthandelExtra   *TrumfNetthandelExtra   `json:"trumfnetthandel_extra,omitempty"`
	SasOnlineShoppingExtra *SasOnlineShoppingExtra `json:"sasonlineshopping_extra,omitempty"`
}

type TrumfNetthandelExtra struct {
	Slug string `json:"slug"`
	Url  string `json:"url"`
}

type SasOnlineShoppingExtra struct {
	UUID string `json:"uuid"`
	Slug string `json:"slug"`
	Url  string `json:"url"`
}

const (
	Trumfnetthandel   = "trumfnetthandel"
	Sasonlineshopping = "sasonlineshopping"
)

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var ip string

        xForwardedFor := r.Header.Get("X-Forwarded-For")
        if xForwardedFor != "" {
            ips := strings.Split(xForwardedFor, ",")
            ip = strings.TrimSpace(ips[0])
        } else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
            ip = realIP
        } else {
            ip = r.RemoteAddr
        }

        userAgent := r.Header.Get("User-Agent")
        event := r.URL.Path

        log.SetFlags(0)
        log.Printf("Request incoming; IP: %s Event: \"%s\" Status: \"%s\" UserAgent:\"%s\"", ip, event, "-", userAgent)

        next(w, r)
    }
}
