package stocks

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dchest/stemmer/porter2"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/language"
	"github.com/itsabot/abot/shared/plugin"
)

var p *dt.Plugin
var regexLetters = regexp.MustCompile(`\W+`)
var regexCapitals = regexp.MustCompile(`[a-z]+`)

func init() {
	var err error
	p, err = plugin.New("github.com/itsabot/plugin_stocks")
	if err != nil {
		log.Fatal(err)
	}
	plugin.SetKeywords(p,
		dt.KeywordHandler{
			Fn: kwGetStockDetails,
			Trigger: &dt.StructuredInput{
				Commands: []string{"what", "show", "tell",
					"how"},
				Objects: []string{"stock", "ticker", "symbol",
					"share"},
			},
		},
	)
	if err = plugin.Register(p); err != nil {
		p.Log.Fatal(err)
	}
}

func kwGetStockDetails(in *dt.Msg) string {
	var stock string
	tickers, err := extractStockTickers(in)
	if err != nil {
		return ""
	}
	stock = tickers[0]
	p.SetMemory(in, "stock", stock)

	// Request stock info from Yahoo
	client := &http.Client{Timeout: 10 * time.Second}
	u := "https://finance.yahoo.com/webservice/v1/symbols/" + stock + "/quote?format=json"
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		p.Log.Info("failed to build request for stock data from Yahoo Finance.", err)
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		p.Log.Info("failed to retrieve stock data from Yahoo Finance.", err)
		return ""
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			p.Log.Info("failed to close Yahoo Finance response body.", err)
		}
	}()
	var stockResp struct {
		List struct {
			Resources []struct {
				Resource struct {
					Fields struct {
						Name   string `json:"name"`
						Price  string `json:"price"`
						Symbol string `json:"symbol"`
					} `json:"fields"`
				} `json:"resource"`
			} `json:"resources"`
		} `json:"list"`
	}
	if resp.StatusCode != 200 {
		p.Log.Info("invalid status code from Yahoo Finance.", resp.StatusCode)
		return ""
	}
	if err = json.NewDecoder(resp.Body).Decode(&stockResp); err != nil {
		p.Log.Info("failed to decode Yahoo Finance response.", err)
		return ""
	}
	if len(stockResp.List.Resources) == 0 {
		return "I couldn't find any ticker symbols matching that."
	}
	return fmt.Sprintf("%s (%s) is trading at %s right now.",
		stockResp.List.Resources[0].Resource.Fields.Name,
		stockResp.List.Resources[0].Resource.Fields.Symbol,
		stockResp.List.Resources[0].Resource.Fields.Price)
}

func extractStockTickers(in *dt.Msg) ([]string, error) {
	var tickers []string
	for _, token := range in.Tokens {
		if len(token) > 5 {
			continue
		}

		// Remove all non-letters, confirm length is still valid
		token = regexLetters.ReplaceAllString(token, "")
		if len(token) == 0 || len(token) > 5 {
			continue
		}

		// If the word is all caps, accept it as a ticker symbol.
		caps := regexCapitals.ReplaceAllString(token, "")
		if len(caps) == len(token) {
			tickers = append(tickers, token)
			continue
		}

		// If the potential ticker symbol is not all caps, but it's not
		// a recognized Command or Object, it's much more likely to be
		// a ticker symbol
		eng := porter2.Stemmer
		stem := eng.Stem(token)
		if language.Contains(in.StructuredInput.Commands, stem) ||
			language.Contains(in.StructuredInput.Objects, stem) {
			continue
		}
		tickers = append(tickers, strings.ToUpper(token))
	}
	if len(tickers) > 0 {
		sort.Sort(sort.Reverse(byLength(tickers)))
		return tickers, nil
	}
	return nil, errors.New("missing ticker symbol")
}

// Sorting functions to sort ticker symbols by length
type byLength []string

func (s byLength) Len() int {
	return len(s)
}
func (s byLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byLength) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}
