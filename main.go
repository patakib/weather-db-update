package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"

	t "weather-db-update/types"
)

func readConfig(file string) (t.Config, error) {
	// read config file and returns the config
	f, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	var config t.Config
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config, err
}

func getMeteoData(coordinates []float64) (t.Response, error) {
	// call open-meteo api - based on config -, and returns a response struct
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%v&longitude=%v&hourly=temperature_2m,precipitation_probability,precipitation,rain,snowfall,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m&daily=temperature_2m_max,temperature_2m_min,sunrise,sunset,precipitation_sum,rain_sum,smowfall_sum,precipitation_hours_precipitation_probability_max,wind_speed_10m_max,wind_direction_10m_dominant&timezone=auto&forecast_days=16", coordinates[0], coordinates[1])

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	responseData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var responseObject t.Response
	json.Unmarshal(responseData, &responseObject)
	return responseObject, err
}

func writeDataToDb(response t.Response, pgPort, pgHost, pgDatabase, pgUser, pgPass, city string) {
	//connects to database and insert data from response struct into db
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", pgHost, pgPort, pgUser, pgPass, pgDatabase)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	createTableHourly := `
		CREATE TABLE IF NOT EXISTS weather_hourly (
			id SERIAL PRIMARY KEY, 
			city VARCHAR(50),
			reg_date DATE,
			date TIMESTAMP, 
			temp_2m DECIMAL, 
			prec_prob INTEGER, 
			prec DECIMAL, 
			rain DECIMAL, 
			snow DECIMAL, 
			cloud_cover INTEGER, 
			windspeed_10m DECIMAL, 
			winddir_10m INTEGER,
			weather_code INTEGER
		);`
	if _, err := db.Exec(createTableHourly); err != nil {
		log.Fatal(err)
	}
	createTableDaily := `
		CREATE TABLE IF NOT EXISTS weather_daily (
			id SERIAL PRIMARY KEY,
			city VARCHAR(50),
			reg_date DATE,
			date DATE,
			sunrise TIMESTAMP,
			sunset TIMESTAMP
		);`
	if _, err := db.Exec(createTableDaily); err != nil {
		log.Fatal(err)
	}
	for index, _ := range response.Hourly.Time {
		insertIntoHourly := `
			INSERT INTO weather_hourly (
				city,
				reg_date,
				date,
				temp_2m,
				prec_prob,
				prec,
				rain,
				snow,
				cloud_cover,
				windspeed_10m,
				winddir_10m,
				weather_code
			)
			VALUES ($1,CURRENT_DATE,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11);`
		if _, err := db.Exec(insertIntoHourly,
			city,
			response.Hourly.Time[index],
			response.Hourly.Temp_2m[index],
			response.Hourly.PrecProb[index],
			response.Hourly.Prec[index],
			response.Hourly.Rain[index],
			response.Hourly.Snow[index],
			response.Hourly.CloudCover[index],
			response.Hourly.Windspeed_10m[index],
			response.Hourly.Winddir_10m[index],
			response.Hourly.WeatherCode[index],
		); err != nil {
			log.Fatal(err)
		}
	}
	for index, _ := range response.Daily.Time {
		insertIntoDaily := `
			INSERT INTO weather_daily (
				city,
				reg_date,
				date,
				sunrise,
				sunset
			)
			VALUES ($1,CURRENT_DATE,$2,$3,$4);`
		if _, err := db.Exec(insertIntoDaily,
			city,
			response.Daily.Time[index],
			response.Daily.Sunrise[index],
			response.Daily.Sunset[index],
		); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Data has been successfully written to Database.")
}

func sortedKeys[V any](m map[int]V) []int {
	//sort keys of a map (with any value type)
	keys := make([]int, 0)
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func createEmailData(response t.Response) t.EmailData {
	//create email data struct from response struct
	var emailData t.EmailData
	emailData.Temperature = make(map[int]float32)
	emailData.Precipitation = make(map[int][]float32)
	emailData.WeatherCode = make(map[int]int)
	for index, _ := range response.Hourly.Time[:24] {
		actualTime, err := time.Parse("2006-01-02T15:04", response.Hourly.Time[index])
		if err != nil {
			panic(err)
		}
		actualHour := actualTime.Hour()
		emailData.Temperature[actualHour] = response.Hourly.Temp_2m[index]
		emailData.Precipitation[actualHour] = append(emailData.Precipitation[actualHour], response.Hourly.Prec[index])
		emailData.Precipitation[actualHour] = append(emailData.Precipitation[actualHour], float32(response.Hourly.PrecProb[index]))
		emailData.WeatherCode[actualHour] = response.Hourly.WeatherCode[index]
	}
	return emailData
}

func writeEmail(emailData t.EmailData, city, user, sender, pass, receiver, host, port string) {
	toAddresses := []string{sender}
	hostAndPort := fmt.Sprintf("%s"+":"+"%s", host, port)
	tempString := ""
	precString := ""
	weatherCodeString := ""
	sortedTemp := sortedKeys(emailData.Temperature)
	sortedPrec := sortedKeys(emailData.Precipitation)
	sortedWeatherCode := sortedKeys(emailData.WeatherCode)
	for indexTemp, _ := range sortedTemp {
		tempString = tempString + fmt.Sprintf("%v ora - %v fok\n", indexTemp, emailData.Temperature[indexTemp])
	}
	for indexPrec, _ := range sortedPrec {
		if emailData.Precipitation[indexPrec][0] > 0 {
			precString = precString + fmt.Sprintf("%v ora - %v mm - %v valoszinuseg\n", indexPrec, emailData.Precipitation[indexPrec][0], emailData.Precipitation[indexPrec][1])
		}
	}
	for indexWeatherCode, _ := range sortedWeatherCode {
		if emailData.WeatherCode[indexWeatherCode] == 99 || emailData.WeatherCode[indexWeatherCode] == 95 || emailData.WeatherCode[indexWeatherCode] == 96 {
			weatherCodeString = weatherCodeString + fmt.Sprintf("%v ora - VIHAR VARHATO!\n", indexWeatherCode)
		}
	}
	msgString := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: Mai idojaras - %s\r\n\r\n"+
			"Homerseklet:\n-----------------\n"+
			"%s\n"+
			"Csapadek:\n-----------------\n"+
			"%s\n"+
			"Viharelorejelzes:\n-------------------\n"+
			"%s\n"+
			"\r\n",
		sender,
		sender,
		city,
		tempString,
		precString,
		weatherCodeString,
	)
	msg := []byte(msgString)
	auth := smtp.PlainAuth("", user, pass, host)
	err := smtp.SendMail(hostAndPort, auth, sender, toAddresses, msg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Email sent successfully.")
}

func main() {
	config, err := readConfig("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err2 := godotenv.Load(".env.secret")
	if err2 != nil {
		log.Fatal()
	}
	db_user := os.Getenv("POSTGRES_USER")
	db_pass := os.Getenv("POSTGRES_PASS")
	db_host := os.Getenv("POSTGRES_HOST")
	db_port := os.Getenv("POSTGRES_PORT")
	db_database := os.Getenv("POSTGRES_DB")
	email_user := os.Getenv("EMAIL_USER")
	email_sender := os.Getenv("EMAIL_SENDER")
	email_sender_pass := os.Getenv("EMAIL_SENDER_PASS")
	receiver := os.Getenv("RECEIVER")
	smtp_host := os.Getenv("SMTP_HOST")
	smtp_port := os.Getenv("SMTP_PORT")

	var wg sync.WaitGroup
	for i, city := range config.Cities {
		wg.Add(1)
		go func(i int, city t.ConfigCity) {
			defer wg.Done()
			res, err := getMeteoData(config.Cities[i].Coordinates, config.Parameters, config.ForecastDays)
			if err != nil {
				log.Fatal(err)
			}
			writeDataToDb(res, db_port, db_host, db_database, db_user, db_pass, city.Name)
			if city.Email == true {
				emailData := createEmailData(res)
				writeEmail(emailData, city.Name, email_user, email_sender, email_sender_pass, receiver, smtp_host, smtp_port)
			}
		}(i, city)
	}
	wg.Wait()
}
