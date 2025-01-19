package main

import (
	"dario.cat/mergo"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Path     string `json:"path"`
	Username string `json:"username"`
	Password string `json:"password"`
	Interval int    `json:"interval"`
	WebPort  int    `json:"webPort"`
	BaseURL  string `json:"baseURL"`
}

const (
	ConfigFileName = "protonwatcher-config.json"
)

var PortRegex = regexp.MustCompile(`pair (\d{5})->\d{5}`)

var RateLimitError = fmt.Errorf("you have been rate limited")
var InvalidCredentialsError = fmt.Errorf("invalid credentials")

func main() {
	// Load configuration
	localAppData, success := os.LookupEnv("LOCALAPPDATA")
	if !success {
		log.Fatalln("LOCALAPPDATA environment variable is not set")
	}

	defaultPath := filepath.Join(localAppData, "ProtonVPN", "Logs", "app-logs.txt")

	defaultConfig := Config{
		Path:     defaultPath,
		Username: "admin",
		Password: "adminadmin",
		Interval: 5000,
		WebPort:  8080,
		BaseURL:  "http://localhost",
	}
	data, err := os.ReadFile(ConfigFileName)

	var config Config

	if errors.Is(err, os.ErrNotExist) {
		log.Println("Config file does not exist, using defaults")
	} else {
		err = json.Unmarshal(data, &config)
		if err != nil {
			log.Println("Config file is not formatted correctly", err)
		}
	}

	// Merge default config with user config
	err = mergo.Merge(&config, defaultConfig)
	if err != nil {
		panic(err)
	}

	configString, _ := json.MarshalIndent(config, "", "\t")
	log.Println("Using configuration\n", string(configString))

	// On startup, run port update immediately
	currentPort := getLatestPort(config)
	log.Println("Detected new port", currentPort)
	lastModifiedTime := getModifiedTime(config)
	err = updatePort(config, currentPort)
	if err != nil {
		if errors.Is(err, RateLimitError) {
			log.Println(err)
		} else if errors.Is(err, InvalidCredentialsError) {
			log.Fatalln(err)
			return
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Println("Updated qBittorrent port", currentPort)
	}

	for {
		time.Sleep(time.Duration(config.Interval) * time.Millisecond)

		modifiedTime := getModifiedTime(config)
		if modifiedTime.Before(lastModifiedTime) {
			continue
		}
		newPort := getLatestPort(config)

		if currentPort == newPort {
			continue
		}

		log.Println("Detected new port", newPort)

		err := updatePort(config, newPort)
		if err != nil {
			if errors.Is(err, RateLimitError) {
				log.Println(err)
			} else if errors.Is(err, InvalidCredentialsError) {
				log.Fatalln(err)
			} else {
				log.Fatalln(err)
			}
		} else {
			lastModifiedTime = modifiedTime
			currentPort = newPort
			log.Println("Updated qBittorrent port", currentPort)
		}

	}
}

func getLatestPort(config Config) string {
	logFile, err := os.ReadFile(config.Path)
	if err != nil {
		panic(err)
	}

	strLogFile := string(logFile)

	results := PortRegex.FindAllStringSubmatch(strLogFile, -1)

	newPort := results[len(results)-1][1]
	return newPort
}

func getModifiedTime(config Config) time.Time {
	info, err := os.Stat(config.Path)
	if err != nil {
		panic(err)
	}
	return info.ModTime()
}

func updatePort(config Config, port string) error {
	urlString := fmt.Sprintf("%s:%d", config.BaseURL, config.WebPort)

	originUrl, err := url.Parse(urlString)
	if err != nil {
		panic(err)
	}

	apiUrl, err := url.JoinPath(originUrl.String(), "/api/v2")
	if err != nil {
		panic(err)
	}

	loginUrl, err := url.JoinPath(apiUrl, "/auth/login")
	if err != nil {
		panic(err)
	}

	loginBody := fmt.Sprintf(`username=%v&password=%v`, config.Username, config.Password)

	loginRequest, err := http.NewRequest("POST", loginUrl, strings.NewReader(loginBody))
	if err != nil {
		panic(err)
	}

	loginRequest.Header = http.Header{
		"Content-Type": {"application/x-www-form-urlencoded"},
		"Referer":      {originUrl.String()},
	}

	client := http.Client{}
	loginResponse, err := client.Do(loginRequest)
	if err != nil {
		panic(err)
	}

	if loginResponse.StatusCode == http.StatusForbidden {
		return RateLimitError
	} else if loginResponse.StatusCode != http.StatusOK {
		return err
	}

	b, err := io.ReadAll(loginResponse.Body)
	if err != nil {
		panic(err)
	}
	body := string(b)

	if body == "Fails." {
		return InvalidCredentialsError
	}

	// Slice cookie at indices 4-36
	setCookie := loginResponse.Header.Get("set-cookie")
	cookie := setCookie[4:36]

	prefsUrl, err := url.JoinPath(apiUrl, "/app/setPreferences")
	if err != nil {
		panic(err)
	}

	prefsBody := fmt.Sprintf(`json={"listen_port": %v}`, port)

	prefsReq, err := http.NewRequest("POST", prefsUrl, strings.NewReader(prefsBody))
	if err != nil {
		panic(err)
	}

	prefsReq.Header = http.Header{
		"Content-Type":    {"application/x-www-form-urlencoded"},
		"Referer":         {originUrl.String()},
		"Accept":          {"*/*"},
		"Accept-Encoding": {"gzip, br"},
	}

	prefsReq.Header.Add("set-cookie", fmt.Sprintf("SID = %s", cookie))

	loginResponse, err = client.Do(prefsReq)
	if err != nil {
		panic(err)
	}
	return nil
}
