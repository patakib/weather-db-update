package types

type Config struct {
	Cities []ConfigCity `yaml:"cities"`
}
type ConfigCity struct {
	Name        string    `yaml:"name"`
	Coordinates []float64 `yaml:"coordinates"`
	Email       bool      `yaml:"email"`
}
type Response struct {
	Hourly HourlyWeather `json:"hourly"`
	Daily  DailyWeather  `json:"daily"`
}
type DailyWeather struct {
	Time                                []string  `json:"time"`
	Temperature2mMax                    []float32 `json:"temperature_2m_max"`
	Temperature2mMin                    []float32 `json:"temperature_2m_min"`
	Sunrise                             []string  `json:"sunrise"`
	Sunset                              []string  `json:"sunset"`
	PrecipitationSum_mm                 []float32 `json:"precipitation_sum"`
	RainSum_mm                          []float32 `json:"rain_sum"`
	SnowfallSum_mm                      []float32 `json:"snowfall_sum"`
	PrecipitationHours_mm               []float32 `json:"precipitation_hours"`
	PrecipitationProbabilityMax_percent []int32   `json:"precipitation_probability_max"`
	WindSpeed10mMax                     []float32 `json:"wind_speed_10m_max"`
	WindDirection10mDominant            []int32   `json:"wind_direction_10m_dominant"`
}
type HourlyWeather struct {
	Time                             []string  `json:"time"`
	Temperature2m                    []float32 `json:"temperature_2m"`
	PrecipitationProbability_percent []int32   `json:"precipitation_probability"`
	Precipitation_mm                 []float32 `json:"precipitation"`
	Rain_mm                          []float32 `json:"rain"`
	Snow_mm                          []float32 `json:"snowfall"`
	WeatherCode                      []int32   `json:"weather_code"`
	CloudCover_percent               []int32   `json:"cloud_cover"`
	WindSpeed10m                     []float32 `json:"wind_speed_10m"`
	WindDirection10m                 []int32   `json:"wind_direction_10m"`
}
