package paymentwall

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const apiBase = "https://api.paymentwall.com/api"

type Product struct {
	ID           string
	Amount       float64
	CurrencyCode string
	Name         string
	Type         string
}

type Widget struct {
	PublicKey  string
	PrivateKey string
	UserID     string
	WidgetCode string
	Product    Product
	Extra      map[string]string
}

func (w Widget) URL() string {
	if w.WidgetCode == "" {
		w.WidgetCode = "p1_1"
	}
	if w.Product.Type == "" {
		w.Product.Type = "fixed"
	}
	params := map[string]string{
		"key":            w.PublicKey,
		"uid":            w.UserID,
		"widget":         w.WidgetCode,
		"amount":         formatAmount(w.Product.Amount),
		"currencyCode":   w.Product.CurrencyCode,
		"ag_name":        w.Product.Name,
		"ag_external_id": w.Product.ID,
		"ag_type":        w.Product.Type,
		"sign_version":   "3",
	}
	for k, v := range w.Extra {
		if v == "" {
			continue
		}
		params[k] = v
	}
	params["sign"] = signV3(params, w.PrivateKey)

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return apiBase + "/subscription?" + q.Encode()
}

func formatAmount(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

func signV3(params map[string]string, privateKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(params[k])
	}
	b.WriteString(privateKey)
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func FormatGoodsID(id any) string {
	return fmt.Sprint(id)
}
