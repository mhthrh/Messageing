package Redis

import (
	"Github.com/mhthrh/EventDriven/Utilitys/Json"
	"fmt"
	"github.com/go-redis/redis"
	"time"
)

type Client struct {
	client *redis.Client
}

func New(host, password string, database, port int) *Client {
	r := new(Client)
	t := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       database,
	})
	r.client = t
	return r
}

func (c *Client) Ping() error {
	_, err := c.client.Ping().Result()
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) Set(Key, values string) error {

	if err := c.client.Set(Key, values, 0).Err(); err != nil {
		return err
	}
	return nil
}

func (c *Client) KeyExist(Key string) (int, error) {
	if err := c.client.Exists(Key).Err(); err != nil {
		return 1, err
	}
	return 0, nil
}
func (c *Client) AddText(Key string, text string, object interface{}) error {
	if r := c.client.Exists(Key); r.Val() != 1 {
		err := c.client.Set(Key, text, 0)
		return err.Err()
	}
	str, err := c.Get(Key)
	if err != nil {
		return err
	}
	return c.Set(Key, fmt.Sprintf("%s\\n%s\\n%s\\n%s", str, text, time.Now(), Json.New(nil, nil).Struct2Json(&object)))

}

func (c *Client) Get(key string) (string, error) {
	Val, err := c.client.Get(key).Result()
	if err != nil {
		return "", err
	}

	return Val, nil
}
