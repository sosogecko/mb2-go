package appcfg

import (
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Store struct {
	db     *gorm.DB
	log    *zap.Logger
	mu     sync.RWMutex
	data   map[string]string
	loaded time.Time
	ttl    time.Duration
}

type row struct {
	Name  string `gorm:"column:name"`
	Value string `gorm:"column:value"`
}

func New(db *gorm.DB, log *zap.Logger) *Store {
	s := &Store{
		db:   db,
		log:  log,
		data: map[string]string{},
		ttl:  60 * time.Second,
	}
	s.reload()
	return s
}

func (s *Store) Get(name string) string {
	s.maybeReload()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.data[name]; ok {
		return v
	}
	for k, v := range s.data {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}

func (s *Store) Int(name string, def int) int {
	v := strings.TrimSpace(s.Get(name))
	if v == "" {
		return def
	}
	n := 0
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	if n == 0 && v != "0" {
		return def
	}
	return n
}

func (s *Store) maybeReload() {
	s.mu.RLock()
	ok := time.Since(s.loaded) < s.ttl
	s.mu.RUnlock()
	if ok {
		return
	}
	s.reload()
}

func (s *Store) reload() {
	var rows []row
	if err := s.db.Table("config").Select("name, value").Find(&rows).Error; err != nil {
		if s.log != nil {
			s.log.Error("load config table failed", zap.Error(err))
		}
		return
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Name] = r.Value
	}
	s.mu.Lock()
	s.data = m
	s.loaded = time.Now()
	s.mu.Unlock()
}
