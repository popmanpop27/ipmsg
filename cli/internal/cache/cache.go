package cache

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type Cache struct {
	FilePath  string    `json:"-"`
	UpdatedAt time.Time `json:"updated_at"`
	IPs       []string  `json:"ips"`

	mu sync.Mutex
}

func New(path string) (*Cache, error) {
	c := &Cache{
		FilePath: path,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := c.write([]string{}); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Cache) GetIps() ([]string, error) {
	cache, err := c.read()
	if err != nil {
		return nil, err
	}
	
	if time.Since(cache.UpdatedAt).Minutes() > 60 {
		return nil, errors.New("cache is expired")
	}

	return cache.IPs, nil
}

// rewriting ips in cache to provided
func (c *Cache) UpdateIps(ips []string) error {
	if err := c.write(ips); err != nil {
		return err
	}

	return nil
}

/* ======== internal ======== */

func (c *Cache) read() (*Cache, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.FilePath)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &Cache{
			FilePath:  c.FilePath,
			IPs:       []string{},
			UpdatedAt: time.Now(),
		}, nil
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	cache.FilePath = c.FilePath
	return &cache, nil
}

func (c *Cache) write(ips []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cache := Cache{
		IPs:       ips,
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}


	file, err := os.OpenFile(c.FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
