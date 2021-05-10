package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

// Store all configuration options.
type Config struct {
	OTRSConnection OTRSConnection `yaml:"OTRSConnection"`
	Web            Web            `yaml:"Web"`
	UserList       []User         `yaml:"UserList"`
}

// Connection to OTRS DB.
type OTRSConnection struct {
	Host     string `yaml:"Host"`
	Port     string `yaml:"Port"`
	UserName string `yaml:"UserName"`
	Password string `yaml:"Password"`
	DBName   string `yaml:"DBName"`
	SSLMode  string `yaml:"SSLMode"`
}

// Web interface.
type Web struct {
	Port string `yaml:"Port"`
}

// Users for whom information is displayed in the web interface.
type User struct {
	LastName  string `yaml:"LastName"`
	WorkShift string `yaml:"WorkShift"`
	Command   int    `yaml:"Command"`
}

// Extract configuration file and unmarshall collected data into config variable.
func ReadConfigFromYAMLFile(cfgFilePath string) (Config, error) {
	log.Println("[START   ] ReadConfigFromYAMLFile")
	file, err := os.Open(cfgFilePath)
	if err != nil {
		log.Println("[FAIL    ] GetCustomizationFoldersList")
		return Config{}, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("[FAIL    ] GetCustomizationFoldersList")
		return Config{}, err
	}
	var mainConfig Config
	err = yaml.Unmarshal(data, &mainConfig)
	if err != nil {
		log.Println("[FAIL    ] GetCustomizationFoldersList")
		return Config{}, err
	}
	log.Println("[SUCCESS ] ReadConfigFromYAMLFile")
	return mainConfig, nil
}
