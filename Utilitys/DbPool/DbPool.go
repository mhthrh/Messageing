package DbPool

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"sync"
	"time"
)

var (
	_driver     string
	_connection string
	_count      int
)

type Connection struct {
	Db      *sql.DB
	working bool
	mu      sync.Mutex
}

type Pool []*Connection

func New(host, pass, dbName, driver string, port, connectionCount int) *Pool {
	_count = connectionCount
	_driver = driver
	_connection = fmt.Sprintf("root:%s@tcp(%s:%d)/%s",
		pass, host, port, dbName)
	var Pool Pool
	for i := 0; i < connectionCount; i++ {
		Pool = append(Pool, &Connection{
			Db:      nil,
			working: false,
		})
	}
	return &Pool
}

func (p *Pool) FillPool() int {
	count := 0
	for i, c := range *p {
		if !c.working {
			if c.Db == nil {
				db, err := sql.Open(_driver, _connection)
				if err != nil {
					continue
				}
				(*p)[i].Db = db
			}

			err := c.Db.Ping()
			if err != nil {
				db, err := sql.Open(_driver, _connection)
				if err != nil {
					continue
				}
				(*p)[i].Db = db
			}
			count++
		}
	}
	return count
}

func (p *Pool) Pull() *Connection {
	for {
		for _, c := range *p {
			c.mu.Lock()
			if !c.working {
				if err := c.Db.Ping(); err == nil {
					c.working = true
					c.mu.Unlock()
					return c
				}
			}
			c.mu.Unlock()
		}
	}

}

func (p *Pool) Refresh(period time.Duration) {
	for range time.Tick(time.Second * period) {
		p.FillPool()
	}

}

func (p *Pool) Push(c *Connection) {
	c.working = false
}
