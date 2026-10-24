package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
)

const maxLogBody = 1 << 20

func Post(c *gin.Context, key string) string {
	if v := c.PostForm(key); v != "" {
		return v
	}
	if v := jsonBody(c)[key]; v != "" {
		return v
	}
	return ""
}

func Get(c *gin.Context, key string) string {
	return c.Query(key)
}

func Param(c *gin.Context, key string) string {
	if v := Post(c, key); v != "" {
		return v
	}
	return Get(c, key)
}

func jsonBody(c *gin.Context) map[string]string {
	cached, ok := c.Get("_json_body_map")
	if ok {
		if m, ok := cached.(map[string]string); ok {
			return m
		}
	}
	m := map[string]string{}
	ct := c.ContentType()
	if strings.Contains(ct, "application/json") && c.Request.Body != nil {
		b, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(strings.NewReader(string(b)))
		var raw map[string]any
		if json.Unmarshal(b, &raw) == nil {
			for k, v := range raw {
				m[k] = fmt.Sprint(v)
			}
		}
	}
	c.Set("_json_body_map", m)
	return m
}

func ClientIP(c *gin.Context) string {
	ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err != nil {
		ip = c.Request.RemoteAddr
	}
	return ip
}

func MD5String(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func PHPMd5Token(email string) string {
	var n int64
	_ = binary.Read(rand.Reader, binary.LittleEndian, &n)
	if n < 0 {
		n = -n
	}
	num := time.Now().Unix() + phpToInt(email) + (n % 32768)
	return MD5String(strconv.FormatInt(num, 10))
}

func phpToInt(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	i := 0
	sign := int64(1)
	if s[0] == '-' {
		sign = -1
		i++
	} else if s[0] == '+' {
		i++
	}
	var n int64
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		n = n*10 + int64(s[i]-'0')
	}
	return n * sign
}

func ParseExpire(expire string, loc *time.Location) int64 {
	expire = strings.TrimSpace(expire)
	if expire == "" {
		return 0
	}
	parts := strings.Split(expire, "-")
	if len(parts) < 3 {
		return 0
	}
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	rest := parts[2]
	day, hour := 0, 0
	if len(rest) >= 2 {
		day, _ = strconv.Atoi(rest[:2])
		if len(rest) > 2 {
			hour, _ = strconv.Atoi(rest[2:])
		}
	}
	if loc == nil {
		loc = time.Local
	}
	return time.Date(year, time.Month(month), day, hour, 0, 0, 0, loc).Unix()
}

func FormatDateTime(ts int64, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}
	return time.Unix(ts, 0).In(loc).Format("2006-01-02 15:04:05")
}

func CRC32Uniq() string {
	var n uint32
	_ = binary.Read(rand.Reader, binary.LittleEndian, &n)
	raw := fmt.Sprintf("%d%x", time.Now().UnixNano(), n)
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(raw)))
}

func VersionCompare(a, b string) int {
	pa := splitVersion(a)
	pb := splitVersion(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		va, vb := 0, 0
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}
	return 0
}

func splitVersion(v string) []int {
	v = strings.TrimSpace(v)
	if v == "" {
		return []int{0}
	}
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || unicode.IsSpace(r)
	})
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(digitsOnly(p))
		out = append(out, n)
	}
	if len(out) == 0 {
		return []int{0}
	}
	return out
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	if b.Len() == 0 {
		return "0"
	}
	return b.String()
}

func IsTruthyOne(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case int:
		return t == 1
	case int64:
		return t == 1
	case float64:
		return t == 1
	case json.Number:
		n, _ := t.Int64()
		return n == 1 || t.String() == "1"
	case string:
		return t == "1"
	case []byte:
		return string(t) == "1"
	default:
		return fmt.Sprint(v) == "1"
	}
}

func StringVal(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprint(v)
	}
}

func MapGetCI(m map[string]any, key string) any {
	if v, ok := m[key]; ok {
		return v
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}

func NormalizeRow(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		return t
	}
}

func Truncate(s string) string {
	if len(s) <= maxLogBody {
		return s
	}
	return s[:maxLogBody] + fmt.Sprintf("...[truncated %d bytes]", len(s)-maxLogBody)
}

func HeaderMap(c *gin.Context) map[string]string {
	out := map[string]string{}
	for k, vs := range c.Request.Header {
		out[k] = strings.Join(vs, ",")
	}
	return out
}
