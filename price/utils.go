package price

import "fmt"

func GetUrl(feeder string, chain_id int, tokenAddr string) string {
	switch feeder {
	case "dexscreener":
		return fmt.Sprintf("https://dexscreener.com/%s/%s", chain_names[chain_id], tokenAddr)
	case "coinmarketcap":
		return fmt.Sprintf("https://coinmarketcap.com/currencies/%s/", tokenAddr)
	}
	return ""
}
