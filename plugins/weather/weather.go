package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// WeatherData 表示天气数据
type WeatherData struct {
	Location struct {
		Name    string  `json:"name"`
		Country string  `json:"country"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		Condition struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
		} `json:"condition"`
		WindKph    float64 `json:"wind_kph"`
		WindDir    string  `json:"wind_dir"`
		Humidity   int     `json:"humidity"`
		Cloud      int     `json:"cloud"`
		FeelsLikeC float64 `json:"feelslike_c"`
	} `json:"current"`
	Forecast struct {
		Forecastday []struct {
			Date string `json:"date"`
			Day  struct {
				MaxtempC     float64 `json:"maxtemp_c"`
				MintempC     float64 `json:"mintemp_c"`
				Condition    struct {
					Text string `json:"text"`
					Icon string `json:"icon"`
				} `json:"condition"`
				TotalprecipMm float64 `json:"totalprecip_mm"`
				MaxwindKph    float64 `json:"maxwind_kph"`
			} `json:"day"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

var (
	apiKey string
)

func init() {
	apiKey = os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		fmt.Println("警告: 未设置 WEATHER_API_KEY 环境变量")
	}
}

// getWeather 获取天气数据
func getWeather(city string) (*WeatherData, error) {
	url := fmt.Sprintf("http://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=3", apiKey, city)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取天气数据失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var data WeatherData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("解析天气数据失败: %v", err)
	}

	return &data, nil
}

// Commands 导出插件命令
var Commands = []*cobra.Command{
	{
		Use:   "weather",
		Short: "查询天气信息",
		Long:  "查询指定城市的当前天气和天气预报",
	},
	{
		Use:   "current [city]",
		Short: "查询当前天气",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			data, err := getWeather(args[0])
			if err != nil {
				fmt.Printf("错误: %v\n", err)
				return
			}

			fmt.Printf("\n%s, %s 当前天气:\n", data.Location.Name, data.Location.Country)
			fmt.Printf("温度: %.1f°C (体感: %.1f°C)\n", data.Current.TempC, data.Current.FeelsLikeC)
			fmt.Printf("天气: %s\n", data.Current.Condition.Text)
			fmt.Printf("风速: %.1f km/h %s\n", data.Current.WindKph, data.Current.WindDir)
			fmt.Printf("湿度: %d%%\n", data.Current.Humidity)
			fmt.Printf("云量: %d%%\n", data.Current.Cloud)
		},
	},
	{
		Use:   "forecast [city]",
		Short: "查询天气预报",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			data, err := getWeather(args[0])
			if err != nil {
				fmt.Printf("错误: %v\n", err)
				return
			}

			fmt.Printf("\n%s, %s 未来3天天气预报:\n", data.Location.Name, data.Location.Country)
			for _, day := range data.Forecast.Forecastday {
				date, _ := time.Parse("2006-01-02", day.Date)
				fmt.Printf("\n%s:\n", date.Format("2006年01月02日"))
				fmt.Printf("最高温度: %.1f°C\n", day.Day.MaxtempC)
				fmt.Printf("最低温度: %.1f°C\n", day.Day.MintempC)
				fmt.Printf("天气: %s\n", day.Day.Condition.Text)
				fmt.Printf("降水量: %.1f mm\n", day.Day.TotalprecipMm)
				fmt.Printf("最大风速: %.1f km/h\n", day.Day.MaxwindKph)
			}
		},
	},
} 