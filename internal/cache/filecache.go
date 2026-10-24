package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileCache struct {
	dir string
	mu  sync.Mutex
}

type entry struct {
	Expire int64           `json:"expire"`
	Value  json.RawMessage `json:"value"`
}

func New(dir string) (*FileCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileCache{dir: dir}, nil
}

func (c *FileCache) filename(key string) string {
	sum := md5.Sum([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".json")
}

func (c *FileCache) Get(key string, dest any) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	b, err := os.ReadFile(c.filename(key))
	if err != nil {
		return false
	}
	var e entry
	if err := json.Unmarshal(b, &e); err != nil {
		return false
	}
	if e.Expire > 0 && time.Now().Unix() > e.Expire {
		_ = os.Remove(c.filename(key))
		return false
	}
	if dest == nil {
		return true
	}
	return json.Unmarshal(e.Value, dest) == nil
}

func (c *FileCache) Set(key string, value any, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	e := entry{Value: raw}
	if ttl > 0 {
		e.Expire = time.Now().Add(ttl).Unix()
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	path := c.filename(key)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		return os.Rename(tmp, path)
	}
	return nil
}

func (c *FileCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = os.Remove(c.filename(key))
}
