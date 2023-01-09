package Config

import (
	"Github.com/mhthrh/EventDriven/Utilitys/CryptoUtil"
	"Github.com/mhthrh/EventDriven/Utilitys/Directory"
	"Github.com/mhthrh/EventDriven/Utilitys/File"
	"Github.com/mhthrh/EventDriven/Utilitys/Json"
)

type DataBase struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	Password string `json:"password"`
	Dbname   string `json:"Dbname"`
	Driver   string `json:"Driver"`
	Count    int    `json:"count"`
	Second   int    `json:"second"`
}
type Redis struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	Password string `json:"password"`
	Database int    `json:"database"`
}
type RabbitMQ struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	UserName string `json:"userName"`
	Password string `json:"password"`
}

type Dispatcher struct {
	ServiceType   int
	ServiceName   string
	InstanceCount int
	Channel       any
}
type Server struct {
	Typ  string
	Name string
	Ip   string
	Port int
}

type Config struct {
	AppName     string       `json:"AppName"`
	IsTest      bool         `json:"IsTest"`
	Version     string       `json:"Version"`
	ExpireDate  string       `json:"ExpireDate"`
	DB          DataBase     `json:"DB"`
	Rabbit      RabbitMQ     `json:"rabbit"`
	Rds         Redis        `json:"rds"`
	Dispatchers []Dispatcher `json:"dispatchers"`
	Listeners   []Server     `json:"listeners"`
}

func ReadConfig(file string) *Config {
	ut := Directory.Ut{}
	d, err := ut.GetPath()
	if err != nil {
		return nil
	}
	var jsonMap *Config

	Json.New(nil, nil).Json2Struct([]byte(func() string {
		k := CryptoUtil.NewKey()
		//temp since GOLAND bug
		d = "E:\\Golang\\EventDriven\\Gateway\\Config"
		file = "\\ConfigCoded"
		//
		k.Text, _ = File.New(d, file).Read()
		dec, _ := k.Decrypt()
		return dec
	}()), &jsonMap)
	return jsonMap
}

func WriteConfig() string {
	cfg := &Config{
		AppName:    "Message",
		IsTest:     true,
		Version:    "1.0.0",
		ExpireDate: "01-01-2024",
		DB: DataBase{
			Host:     "localhost",
			Port:     3306,
			Password: "123456",
			Dbname:   "MyDb",
			Driver:   "mysql",
			Count:    10,
			Second:   300,
		},
		Listeners: []Server{
			{
				Typ:  "Sync",
				Name: "Server1",
				Ip:   "127.0.0.1",
				Port: 8585,
			},
			{
				Typ:  "Sync",
				Name: "Server2",
				Ip:   "127.0.0.1",
				Port: 8586,
			},
			{
				Typ:  "Sync",
				Name: "Server3",
				Ip:   "127.0.0.1",
				Port: 8587,
			},
			{
				Typ:  "Async",
				Name: "Server4",
				Ip:   "127.0.0.1",
				Port: 8588,
			},
			{
				Typ:  "Async",
				Name: "Server5",
				Ip:   "127.0.0.1",
				Port: 8589,
			},
			{
				Typ:  "Async",
				Name: "Server5",
				Ip:   "127.0.0.1",
				Port: 8590,
			},
			{
				Typ:  "Async",
				Name: "Server5",
				Ip:   "127.0.0.1",
				Port: 8591,
			},
			{
				Typ:  "Async",
				Name: "Server5",
				Ip:   "127.0.0.1",
				Port: 8592,
			},
			{
				Typ:  "Async",
				Name: "Server5",
				Ip:   "127.0.0.1",
				Port: 8593,
			},
		},
		Rabbit: RabbitMQ{
			Host:     "localhost",
			Port:     5672,
			UserName: "guest",
			Password: "guest",
		},
		Rds: Redis{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			Database: 0,
		},
		Dispatchers: []Dispatcher{{
			ServiceType:   1,
			ServiceName:   "Authenticate",
			InstanceCount: 1,
			Channel:       nil,
		}, {
			ServiceType:   2,
			ServiceName:   "Messaging",
			InstanceCount: 1,
			Channel:       nil,
		}},
	}
	s := Json.New(nil, nil).Struct2Json(cfg)
	k := CryptoUtil.NewKey()
	k.Text = s
	enc, _ := k.Encrypt()
	return enc

}
