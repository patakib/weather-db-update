package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"

	t "weather-db-update/types"
)

var infoLogger *log.Logger
var errorLogger *log.Logger

func init() {
	infoLogger = log.New(os.Stdout, "Wweather-db-update - INFO: ", log.LstdFlags|log.Lshortfile)
	errorLogger = log.New(os.Stdout, "weather-db-update - ERROR: ", log.LstdFlags|log.Lshortfile)
}

func readConfig(file string) (t.Config, error) {
	// read config file and returns the config
	f, err := os.ReadFile(file)
	if err != nil {
		errorLogger.Fatalf("Cannot read config: %v", err)
	}
	infoLogger.Println("Config file successfully read.")
	var config t.Config
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		errorLogger.Fatalf("Cannot unmarshal config data: %v", err)
	}
	infoLogger.Println("YAML data has been successfully unmarshalled.")
	return config, err
}

func getMeteoData(coordinates []float64) (t.Response, error) {
	// calls open-meteo api - based on config -, and returns a response struct

	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%v&longitude=%v&hourly=temperature_2m,precipitation_probability,precipitation,rain,snowfall,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m&daily=temperature_2m_max,temperature_2m_min,sunrise,sunset,precipitation_sum,rain_sum,snowfall_sum,precipitation_hours,precipitation_probability_max,wind_speed_10m_max,wind_direction_10m_dominant&timezone=auto&forecast_days=16", coordinates[0], coordinates[1])
	res, err := http.Get(url)
	if err != nil {
		errorLogger.Fatalf("Cannot get meteo data from path: %v\n, %v", url, err)
	}
	responseData, err := io.ReadAll(res.Body)
	if err != nil {
		errorLogger.Fatalf("Cannot read HTTP Response from meteo API: %v", err)
	}
	var responseObject t.Response
	err = json.Unmarshal(responseData, &responseObject)
	if err != nil {
		errorLogger.Fatalf("Cannot unmarshal REST API data: %v\n, %v", responseData, err)
	}
	infoLogger.Println("Meteo data has been successfully extracted.")
	return responseObject, err
}

func writeDataToDb(response t.Response, pgPort, pgHost, pgDatabase, pgUser, pgPass, city string) {
	//connects to database and insert data from response struct into db

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", pgHost, pgPort, pgUser, pgPass, pgDatabase)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		errorLogger.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		errorLogger.Fatal(err)
	}
	createTableHourly := `
		CREATE TABLE IF NOT EXISTS weather_hourly_forecast (
			city VARCHAR(50),
			forecast_date DATE,
			time TIMESTAMP,
			temperature_2m DECIMAL,
			precipitation_probability_percent INTEGER, 
			precipitation_mm DECIMAL, 
			rain_mm DECIMAL, 
			snowfall_cm DECIMAL, 
			weather_code INTEGER,
			cloud_cover_percent INTEGER, 
			windspeed_10m DECIMAL, 
			winddir_10m INTEGER
		);`
	if _, err := db.Exec(createTableHourly); err != nil {
		errorLogger.Fatal(err)
	}
	createTableDaily := `
		CREATE TABLE IF NOT EXISTS weather_daily_forecast (
			city VARCHAR(50),
			forecast_date DATE,
			time DATE,
			temperature_2m_max DECIMAL,
			temperature_2m_min DECIMAL,
			sunrise TIMESTAMP,
			sunset TIMESTAMP,
			precipitation_sum_mm DECIMAL,
			rain_sum_mm DECIMAL,
			snowfall_sum_cm DECIMAL,
			precipitation_hours DECIMAL,
			precipitation_probability_max INTEGER,
			windspeed_10m_max DECIMAL,
			winddirection_10m_dominant INTEGER
		);`
	if _, err := db.Exec(createTableDaily); err != nil {
		errorLogger.Fatal(err)
	}

	if _, err := db.Exec(`DELETE FROM weather_hourly_forecast WHERE city IS NOT NULL;`); err != nil {
		errorLogger.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM weather_daily_forecast WHERE city IS NOT NULL;`); err != nil {
		errorLogger.Fatal(err)
	}

	for index, _ := range response.Hourly.Time {
		insertIntoHourly := `
			INSERT INTO weather_hourly_forecast (
				city,
				forecast_date,
				time,
				temperature_2m,
				precipitation_probability_percent,
				precipitation_mm,
				rain_mm,
				snowfall_cm,
				weather_code,
				cloud_cover_percent,
				windspeed_10m,
				winddir_10m
			)
			VALUES ($1,CURRENT_DATE,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11);`
		if _, err := db.Exec(insertIntoHourly,
			city,
			response.Hourly.Time[index],
			response.Hourly.Temperature2m[index],
			response.Hourly.PrecipitationProbability_percent[index],
			response.Hourly.Precipitation_mm[index],
			response.Hourly.Rain_mm[index],
			response.Hourly.Snow_cm[index],
			response.Hourly.WeatherCode[index],
			response.Hourly.CloudCover_percent[index],
			response.Hourly.WindSpeed10m[index],
			response.Hourly.WindDirection10m[index],
		); err != nil {
			errorLogger.Fatal(err)
		}
	}
	for index, _ := range response.Daily.Time {
		insertIntoDaily := `
			INSERT INTO weather_daily_forecast (
				city,
				forecast_date,
				time,
				temperature_2m_max,
				temperature_2m_min,
				sunrise,
				sunset,
				precipitation_sum_mm,
				rain_sum_mm,
				snowfall_sum_cm,
				precipitation_hours,
				precipitation_probability_max,
				windspeed_10m_max,
				winddirection_10m_dominant
			)
			VALUES ($1,CURRENT_DATE,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);`
		if _, err := db.Exec(insertIntoDaily,
			city,
			response.Daily.Time[index],
			response.Daily.Temperature2mMax[index],
			response.Daily.Temperature2mMin[index],
			response.Daily.Sunrise[index],
			response.Daily.Sunset[index],
			response.Daily.PrecipitationSum_mm[index],
			response.Daily.RainSum_mm[index],
			response.Daily.SnowfallSum_mm[index],
			response.Daily.PrecipitationHours_mm[index],
			response.Daily.PrecipitationProbabilityMax_percent[index],
			response.Daily.WindSpeed10mMax[index],
			response.Daily.WindDirection10mDominant[index],
		); err != nil {
			errorLogger.Fatal(err)
		}
	}
	fmt.Println("Data has been successfully written to Database.")
}

func main() {
	config, err := readConfig("config.yaml")
	if err != nil {
		errorLogger.Fatal(err)
	}
	err2 := godotenv.Load(".env.secret")
	if err2 != nil {
		errorLogger.Fatal()
	}
	db_user := os.Getenv("POSTGRES_USER")
	db_pass := os.Getenv("POSTGRES_PASSWORD")
	db_host := os.Getenv("POSTGRES_HOST")
	db_port := os.Getenv("POSTGRES_PORT")
	db_database := os.Getenv("POSTGRES_DB")

	for i, city := range config.Cities {
		res, err := getMeteoData(config.Cities[i].Coordinates)
		if err != nil {
			errorLogger.Fatal(err)
		}
		writeDataToDb(res, db_port, db_host, db_database, db_user, db_pass, city.Name)
	}
}
