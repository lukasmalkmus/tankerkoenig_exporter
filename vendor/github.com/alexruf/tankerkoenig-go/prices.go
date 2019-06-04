package tankerkoenig

import (
	"fmt"
	"net/url"
	"strings"
)

// PricesService is an interface to query price information from the Tankerkönig-API.
type PricesService interface {
	Get(ids []string) (map[string]Price, *Response, error)
}

// PricesServiceOp handles communication with the price related methods of the Tankerkönig-API.
type PricesServiceOp struct {
	client *Client
}

var _ PricesService = &PricesServiceOp{}

// Price represents a price data structure.
type Price struct {
	Status string      `json:"status"` // Open-status
	Diesel interface{} `json:"diesel"` // Price for diesel fuel type
	E5     interface{} `json:"e5"`     // Price for E5 fuel type
	E10    interface{} `json:"e10"`    // Price for E10 fuel type
}

// pricesRoot represents a response from the Tankerkönig-API.
type pricesRoot struct {
	Ok      bool             `json:"ok"`
	License string           `json:"license"`
	Data    string           `json:"data"`
	Prices  map[string]Price `json:"prices"`
}

// Get returns a map of Price items for a list station IDs.
func (p *PricesServiceOp) Get(ids []string) (map[string]Price, *Response, error) {
	path := "json/prices.php"

	if ids != nil {
		for n, id := range ids {
			ids[n] = fmt.Sprintf("%q", id)
		}
	}

	query := url.Values{}
	query.Add("ids", fmt.Sprintf("[%s]", strings.Join(ids, ",")))
	query.Add("apikey", p.client.APIKey)

	req, err := p.client.NewRequest("GET", path, query, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(pricesRoot)
	resp, err := p.client.Do(req, root)
	if err != nil {
		return nil, nil, err
	}

	return root.Prices, resp, nil
}
