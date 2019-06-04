# tankerkoenig-go

A wrapper for the [Tankerkönig-API](https://creativecommons.tankerkoenig.de/) in Go

## Usage

```go
import "github.com/alexruf/tankerkoenig-go"
```
Create a new client, then use the exposed services to access the API.

## Example

_Note that the Demo-API-Key won't return any actual data results! So when using the API make sure to [register](https://creativecommons.tankerkoenig.de/#register) for a real API-Key._

_The API is under Creative Commons (CC BY 4.0) license. It is used by many clients so please restrict API calls to the minimum. An API call every 15 minutes is OK._

To return a list of stations within a radius of a specific location:

```go
package main

import (
	"fmt"
	"github.com/alexruf/tankerkoenig-go"
	"time"
)

func main() {
	client := tankerkoenig.NewClient("00000000-0000-0000-0000-000000000002", nil)
	stations, _, err := client.Station.List(52.52099975265203, 13.43803882598877, 4)

	if err != nil {
		fmt.Printf("Something bad happened: %s\n\n", err)
		return
	}

	fmt.Printf("Prices %s\n\n", time.Now().Format(time.RFC822))
	for _, station := range stations {
		fmt.Printf("Brand: %s\n", station.Brand)
		fmt.Printf("Name: %s\n", station.Name)
		fmt.Printf("Adress: %s %s, %d %s\n", station.Street, station.HouseNumber, station.PostCode, station.Place)
		fmt.Printf("Diesel:\t%f EUR/l\n", station.Diesel)
		fmt.Printf("E5:\t%f EUR/l\n", station.E5)
		fmt.Printf("E10:\t%f EUR/l\n\n", station.E10)
	}
}
```

To get price information for one or more known gas stations:

```go
package main

import (
	"fmt"
	"github.com/alexruf/tankerkoenig-go"
	"time"
)

func main() {
	client := tankerkoenig.NewClient("00000000-0000-0000-0000-000000000002", nil)

	ids := []string{"1c4f126b-1f3c-4b38-9692-05c400ea8e61", "51d4b6a9-a095-1aa0-e100-80009459e03a", "579d25fd-acb9-445a-9494-f7fe0fa7ce4a", "51d4b660-a095-1aa0-e100-80009459e03a"}
	prices, _, err := client.Prices.Get(ids)

	if err != nil {
		fmt.Printf("Something bad happened: %s\n\n", err)
		return
	}

	fmt.Printf("Prices %s\n\n", time.Now().Format(time.RFC822))
	for id, price := range prices {
		fmt.Println(id)
		fmt.Printf("Status: %s\n", price.Status)
		fmt.Printf("Diesel:\t%v EUR/l\n", price.Diesel)
		fmt.Printf("E5:\t%v EUR/l\n", price.E5)
		fmt.Printf("E10:\t%v EUR/l\n\n", price.E10)
	}
}
```
