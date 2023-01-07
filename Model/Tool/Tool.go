package Tool

import (
	"Github.com/mhthrh/EventDriven/Utilitys/Config"
	"Github.com/mhthrh/EventDriven/Utilitys/DbPool"
	"Github.com/mhthrh/EventDriven/Utilitys/RabbitMQ"
	"Github.com/mhthrh/EventDriven/Utilitys/Redis"
	"Github.com/mhthrh/EventDriven/Utilitys/TextColors"
	"Github.com/mhthrh/EventDriven/Utilitys/Validation"
	"fmt"
)

type Tool struct {
	DB         *DbPool.Pool
	Rabbit     *RabbitMQ.RabbitMq
	Redis      *Redis.Client
	Validation *Validation.Validation
	Config     *Config.Config
}

func New(typ int) (*Tool, error) {
	defer fmt.Println((*TextColors.AllColors())["reset"])

	switch typ {
	case 1:
		fmt.Println((*TextColors.AllColors())["Green"])
	case 2:
		fmt.Println((*TextColors.AllColors())["Blue"])
	case 3:
		fmt.Println((*TextColors.AllColors())["Yellow"])
	default:
		fmt.Println((*TextColors.AllColors())["reset"])
	}

	fmt.Println("Start reading config file")
	cfg := Config.ReadConfig("Config/ConfigCoded")
	if cfg == nil {
		return nil, fmt.Errorf("cant read Config,By")
	}

	rabbit, err := RabbitMQ.New(fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.Rabbit.UserName, cfg.Rabbit.Password, cfg.Rabbit.Host, cfg.Rabbit.Port), cfg)
	if err != nil {
		return nil, fmt.Errorf("cant start rabbitMQ")
	}

	redis := Redis.New(cfg.Rds.Host, cfg.Rds.Password, cfg.Rds.Database, cfg.Rds.Port)
	if err := redis.Ping(); err != nil {
		return nil, fmt.Errorf("redis is not reachable")
	}

	db := DbPool.New(cfg.DB.Host, cfg.DB.Password, cfg.DB.Dbname, cfg.DB.Driver, cfg.DB.Port, cfg.DB.Count)
	count := db.FillPool()
	if count != cfg.DB.Count {
		return nil, fmt.Errorf("cant start Database")
	}
	fmt.Println("Resource initialized successful")
	return &Tool{
		DB:         db,
		Rabbit:     rabbit,
		Redis:      redis,
		Validation: Validation.NewValidation(),
		Config:     cfg,
	}, nil
}
