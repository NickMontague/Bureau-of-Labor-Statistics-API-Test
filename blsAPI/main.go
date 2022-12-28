package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-echarts/go-echarts/charts"
	"github.com/sirupsen/logrus"
)

var (
	filename    = "output.html"
	numElements = 50
	port        = ":8080"
	wg          sync.WaitGroup

	years []string

	unemploymentAverage []float64
	unemploymentChange  []float64

	compensationAverage []float64
	compenstationChange []float64
)

type Request struct {
	Seriesid        []string `json:"seriesid"`
	Startyear       string   `json:"startyear"`
	Endyear         string   `json:"endyear"`
	RegistrationKey string   `json:"registrationkey"`
}

type Response struct {
	Status       string        `json:"status"`
	ResponseTime int           `json:"responseTime"`
	Message      []interface{} `json:"message"`
	Results      struct {
		Series []struct {
			SeriesID string `json:"seriesID"`
			Data     []struct {
				Year       string `json:"year"`
				Period     string `json:"period"`
				PeriodName string `json:"periodName"`
				Latest     string `json:"latest,omitempty"`
				Value      string `json:"value"`
				Footnotes  []struct {
				} `json:"footnotes"`
			} `json:"data"`
		} `json:"series"`
	} `json:"Results"`
}

func main() {

	getUnemployment()
	getCompensation()

	years = append(years, "2015", "2016", "2017", "2018", "2019", "2020", "2021")

	formatUnemployment()
	formatCompensation()

	flag.StringVar(&filename, "o", filename, "Output filename (html)")
	flag.IntVar(&numElements, "c", numElements, "Number of data points")
	flag.Parse()

	startWebServer()

	hostname, _ := os.Hostname()
	logrus.Infof("Listing on http://%v%v", hostname, port)
	wg.Wait()

}

func getUnemployment() {

	var seriesID []string
	// string_first = append(string_first, "LNU04000000") // Annual Unemployment
	seriesID = append(seriesID, "LNS14000000") // Monthly Unemployment

	request := Request{
		Seriesid:        seriesID,
		Startyear:       "2015",
		Endyear:         "2021",
		RegistrationKey: "7598c2d20c3f4a73b3e447af4a2889b7",
	}
	body, _ := json.Marshal(request)

	resp, err := http.Post("https://api.bls.gov/publicAPI/v2/timeseries/data/", "application/json", bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var responseObject Response
	json.Unmarshal(responseData, &responseObject)

	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "./mydatabase.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	var tableExists int
	err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='unemployment'").Scan(&tableExists)
	if err != nil {
		fmt.Println(err)
		return
	}

	// If the "unemployment" table does not exist, create it
	if tableExists == 0 {
		_, err = db.Exec("CREATE TABLE unemployment (year INTEGER PRIMARY KEY, January INTEGER, February INTEGER, March INTEGER, April INTEGER, May INTEGER, June INTEGER, July INTEGER, August INTEGER, September INTEGER, October INTEGER, November INTEGER, December INTEGER)")
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for _, v := range responseObject.Results.Series {
		for _, v := range v.Data {

			// Check if a row with the specified primary key exists
			var rowExists int
			err = db.QueryRow("SELECT count(*) FROM unemployment WHERE year = ?", v.Year).Scan(&rowExists)
			if err != nil {
				fmt.Println(err)
				return
			}

			// If the row does not exist, insert it into the "unemployment" table
			if rowExists == 0 {
				_, err = db.Exec(`INSERT INTO unemployment (year) VALUES (?)`, v.Year)
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			// Update the year and month with a new value
			s := fmt.Sprintf(`UPDATE unemployment SET %s = ? WHERE year = ?`, v.PeriodName)
			_, err = db.Exec(s, v.Value, v.Year)
			if err != nil {
				fmt.Println(err)
				return
			}

		}

	}

}

func getCompensation() {

	var seriesID []string
	seriesID = append(seriesID, "CMU2010000000000D") //  Private industry, All workers, Total compensation - CMU2010000000000D
	// Civilian, All workers, Total compensation - CMU1010000000000D
	// State and local government, All workers, Total compensation - CMU3010000000000D

	request := Request{
		Seriesid:        seriesID,
		Startyear:       "2015",
		Endyear:         "2021",
		RegistrationKey: "7598c2d20c3f4a73b3e447af4a2889b7",
	}

	body, _ := json.Marshal(request)

	resp, err := http.Post("https://api.bls.gov/publicAPI/v2/timeseries/data/", "application/json", bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var responseObject Response
	json.Unmarshal(responseData, &responseObject)

	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "./mydatabase.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	var tableExists int
	err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='compensation'").Scan(&tableExists)
	if err != nil {
		fmt.Println(err)
		return
	}

	// If the "compensation" table does not exist, create it
	if tableExists == 0 {
		_, err = db.Exec("CREATE TABLE compensation (year INTEGER PRIMARY KEY, Q01 INTEGER, Q02 INTEGER, Q03 INTEGER, Q04 INTEGER)")
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for _, v := range responseObject.Results.Series {
		for _, v := range v.Data {

			// Check if a row with the specified primary key exists
			var rowExists int
			err = db.QueryRow("SELECT count(*) FROM compensation WHERE year = ?", v.Year).Scan(&rowExists)
			if err != nil {
				fmt.Println(err)
				return
			}

			// If the row does not exist, insert it into the "compensation" table
			if rowExists == 0 {
				_, err = db.Exec(`INSERT INTO compensation (year) VALUES (?)`, v.Year)
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			// Update the year and quarter with a new value
			s := fmt.Sprintf(`UPDATE compensation SET %s = ? WHERE year = ?`, v.Period)
			_, err = db.Exec(s, v.Value, v.Year)
			if err != nil {
				fmt.Println(err)
				return
			}

		}

	}

}

func startWebServer() {
	wg.Add(1)
	go func() {

		http.HandleFunc("/", renderPage)
		http.ListenAndServe(port, nil)
		wg.Done()
	}()
}

func renderPage(w http.ResponseWriter, r *http.Request) {
	chart1 := createChart1()
	//chart2 := createChart2()

	page := charts.NewPage()
	page.Add(chart1)
	//page.Add(chart2)

	err := page.Render(w)
	if err != nil {
		logrus.Errorf("Unable to render graph: %v", err)
		return
	}
}

func createChart1() *charts.Line {
	line := charts.NewLine()
	line.AddXAxis(years[1:7])
	line.AddYAxis("Unemployment", unemploymentChange, charts.LineOpts{Smooth: true})
	line.AddYAxis("Compensation", compenstationChange, charts.LineOpts{Smooth: true})
	line.Title = "Compensation vs Unemployemnt before and after COVID-19"
	return line
}

func formatUnemployment() {

	// Open the database connection
	db, err := sql.Open("sqlite3", "./mydatabase.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	for _, year := range years {

		// Create the SELECT statement
		sqlStmt := `SELECT January, February, March, April, May, June, July, August, September, October, November, December FROM 'unemployment' WHERE year=?`

		// Execute the SELECT statement
		rows, err := db.Query(sqlStmt, year)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer rows.Close()

		// Create an empty slice of float64
		var values []float64

		// Iterate over the rows and store the values in the slice
		for rows.Next() {
			var January float64
			var February float64
			var March float64
			var April float64
			var May float64
			var June float64
			var July float64
			var August float64
			var September float64
			var October float64
			var November float64
			var December float64
			if err := rows.Scan(&January, &February, &March, &April, &May, &June, &July, &August, &September, &October, &November, &December); err != nil {
				fmt.Println(err)
				return
			}
			values = append(values, January, February, March, April, May, June, July, August, September, October, November, December)
		}

		// Initialize a variable to store the sum
		var sum float64 = 0

		// Iterate over the slice and add each value to the sum
		for _, value := range values {
			sum += value
		}

		// Calculate the average
		average := sum / float64(len(values))

		unemploymentAverage = append(unemploymentAverage, average)
	}

	// Iterate over the slice and calculate the percent change from each index
	for i, value := range unemploymentAverage {
		if i > 0 {
			previousValue := unemploymentAverage[i-1]
			percentChange := (value - previousValue) / previousValue * 100

			unemploymentChange = append(unemploymentChange, percentChange)
		}
	}
}

func formatCompensation() {

	// Open the database connection
	db, err := sql.Open("sqlite3", "./mydatabase.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	for _, year := range years {

		// Create the SELECT statement
		sqlStmt := `SELECT Q01, Q02, Q03, Q04 FROM 'compensation' WHERE year=?`

		// Execute the SELECT statement
		rows, err := db.Query(sqlStmt, year)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer rows.Close()

		// Create an empty slice of integers
		var values []float64

		// Iterate over the rows and store the values in the slice
		for rows.Next() {
			var Q01 float64
			var Q02 float64
			var Q03 float64
			var Q04 float64
			if err := rows.Scan(&Q01, &Q02, &Q03, &Q04); err != nil {
				fmt.Println(err)
				return
			}
			values = append(values, Q01, Q02, Q03, Q04)
		}

		// Initialize a variable to store the sum
		var sum float64 = 0

		// Iterate over the slice and add each value to the sum
		for _, value := range values {
			sum += value
		}

		// Calculate the average
		average := sum / float64(len(values))

		compensationAverage = append(compensationAverage, average)
	}

	// Iterate over the slice and calculate the percent change from each index
	for i, value := range compensationAverage {
		if i > 0 {
			previousValue := compensationAverage[i-1]
			percentChange := (value - previousValue) / previousValue * 100

			compenstationChange = append(compenstationChange, percentChange)
		}
	}
}
