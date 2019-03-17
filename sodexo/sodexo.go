package sodexo

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

var (
	MenuJSONRegex = regexp.MustCompile(`var\s*nd\s*=\s*(.*\])\s*;\n`)
)

func GetTodaysMenu() (MenuDay, error) {
	resp, err := http.Get("https://menus.sodexomyway.com/BiteMenu/Menu?menuId=15109&locationId=10344001&whereami=http://dsu.sodexomyway.com/dining-near-me/trojan-marketplace")

	if err != nil {
		return MenuDay{}, err
	}

	document, err := htmlquery.Parse(resp.Body)

	if err != nil {
		return MenuDay{}, err
	}

	informationScript := htmlquery.FindOne(document, `//script[contains(text(), "var nd")]/text()`)

	// fmt.Println(informationScript.Data)

	ParsedMenuJSON := MenuJSONRegex.FindStringSubmatch(informationScript.Data)

	if len(ParsedMenuJSON) == 0 {
		return MenuDay{}, errors.New("could not find data in response")
	}

	var (
		MenuDate  time.Time
		MenusJSON []MenuDay
	)
	// fmt.Println(ParsedMenuJSON[0])

	err = json.NewDecoder(strings.NewReader(ParsedMenuJSON[1])).Decode(&MenusJSON)

	currentDate := time.Now()

	for _, menuJSON := range MenusJSON {
		MenuDate, err = time.Parse("2006-01-02T15:04:05", menuJSON.Date)

		if err == nil && currentDate.Year() == MenuDate.Year() && currentDate.YearDay() == MenuDate.YearDay() {
			return menuJSON, nil
		}
	}

	return MenuDay{}, errors.New("could not find the menu information for today")
}
