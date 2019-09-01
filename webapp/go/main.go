package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	// "sync"
)

var db *sql.DB

// var mu sync.Mutex

type CarInformation struct {
	Date       time.Time   `json:"date"`
	TrainClass string      `json:"train_class"`
	TrainName  string      `json:"train_name"`
	CarNumber  int         `json:"car_number"`
	SeatList   []TrainSeat `json:"seats"`
}

type Train struct {
	Class string `json:"train_class"`
	Name  string `json:"train_name"`
	Start string `json:"start"`
	Last  string `json:"last"`
}

type TrainSeat struct {
	Row           int    `json:"row"`
	Column        string `json:"column"`
	Class         string `json:"class"`
	IsSmokingSeat bool   `json:"is_smoking_seat"`
	IsOccupied    bool   `json:"is_occupied"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

func distanceFareHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM distance_fare_master")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var distance int
	var fare int
	for rows.Next() {
		err := rows.Scan(&distance, &fare)
		if err != nil {
			panic(err)
		}

		fmt.Fprintf(w, "%d,%d\n", distance, fare)
	}

	err = rows.Err()
	if err != nil {
		panic(err)
	}
}

/*
func fare_calc(date time.Time, depStation, destStation, trainClass, seatClass string)
{
	//
		// 料金計算メモ
		// 距離運賃(円) * 期間倍率(繁忙期なら2倍等) * 車両クラス倍率(急行・各停等) * 座席クラス倍率(プレミアム・指定席・自由席)
	//


	rows, err := db.Query("SELECT * FROM fare_master")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var tc int
	var sc int
	var d time.Time
	var m float
	for rows.Next() {
		err := rows.Scan(&tc, &sc, &d, &m)
		if err != nil {
			panic(err)
		}

		// if

		fmt.Fprintf(w, "1234\n")
	}

	err = rows.Err()
	if err != nil {
		panic(err)
	}
}
*/

func trainSearchHandler(w http.ResponseWriter, r *http.Request) {
	/*
		列車検索
			GET /train/search?use_at=<ISO8601形式の時刻> & from=東京 & to=大阪

		return
			料金
			空席情報
			発駅と着駅の到着時刻

	*/

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	date, err := time.Parse(time.RFC3339, r.URL.Query().Get("use_at"))
	if err != nil {
		panic(err)
	}
	date = date.In(jst)

	trainClass := r.URL.Query().Get("train_class")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	rows, err := db.Query("SELECT departure_at,train_class,train_name,start_station,last_station FROM train_master WHERE date=?",
		date.Format("2006-01-02"))
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// var departureAt time.Time
	var departureAt, trainName, startStation, lastStation string
	trainList := []TrainSearchResponse{}
	for rows.Next() {
		err := rows.Scan(&departureAt, &trainClass, &trainName, &startStation, &lastStation)
		if err != nil {
			panic(err)
		}

		var fromStationAt, toStationAt float64
		db.QueryRow("SELECT distance FROM station_master WHERE name=?", from).Scan(&fromStationAt)
		db.QueryRow("SELECT distance FROM station_master WHERE name=?", to).Scan(&toStationAt)

		// fmt.Println(from_station_at)
		// fmt.Println(to_station_at)

		query := "SELECT name FROM station_master ORDER BY distance"
		if fromStationAt > toStationAt {
			// 上りだったら駅リストを逆にする
			query += " DESC"
		}
		stations, err := db.Query(query)
		if err != nil {
			panic(err)
		}
		isSeekedToFirstStation := false
		isContainsOriginStation := false
		isContainsDestStation := false
		i := 0
		for stations.Next() {
			var v string
			stations.Scan(&v)
			// fmt.Println(v)

			if !isSeekedToFirstStation {
				// 駅リストを列車の発駅まで読み飛ばして頭出しをする
				// 列車の発駅以前は止まらないので無視して良い
				if v == startStation {
					isSeekedToFirstStation = true
				} else {
					continue
				}
			}

			if v == from {
				// 発駅を経路中に持つ編成の場合フラグを立てる
				isContainsOriginStation = true
				fmt.Println(v)
			}
			if v == to {
				if isContainsOriginStation {
					// 発駅と着駅を経路中に持つ編成の場合
					fmt.Println(v)
					fmt.Println("---------")
					isContainsDestStation = true
					break
				} else {
					// 出発駅より先に終点が見つかったとき
					// 上り対応したら要らなくなる
					fmt.Println("なんかおかしい")
					fmt.Println("---------")
					break
				}
			}
			if v == lastStation {
				// 駅が見つからないまま当該編成の終点に着いてしまったとき
				break
			}
			i++
		}
		stations.Close()
		if isContainsOriginStation && isContainsDestStation {
			// TODO: 料金計算
			// TODO: 空席情報
			// TODO: 所要時間計算
			trainList = append(trainList, Train{trainClass, trainName, startStation, lastStation})
		}
	}
	resp, err := json.Marshal(trainList)
	if err != nil {
		panic(err)
	}
	w.Write(resp)

	err = rows.Err()
	if err != nil {
		panic(err)
	}
}

func trainSeatsHandler(w http.ResponseWriter, r *http.Request) {
	/*
		指定した列車の座席列挙
		GET /train/seats?train_class=のぞみ && train_name=96号
	*/

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	date, err := time.Parse(time.RFC3339, r.URL.Query().Get("use_at"))
	if err != nil {
		panic(err)
	}
	date = date.In(jst)

	trainClass := r.URL.Query().Get("train_class")
	trainName := r.URL.Query().Get("train_name")
	carNumber, err := strconv.Atoi(r.URL.Query().Get("car_num"))
	if err != nil {
		panic(err)
	}

	rows, err := db.Query("SELECT seat_column,seat_row,seat_class,is_smoking_seat FROM seat_master WHERE train_class=? AND car_number=?",
		trainClass, carNumber)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var seatRow int
	var seatColumn, seatClass string
	var isSmokingSeat bool
	var seats []TrainSeat
	for rows.Next() {
		err := rows.Scan(&seatColumn, &seatRow, &seatClass, &isSmokingSeat)
		if err != nil {
			panic(err)
		}
		var result int
		db.QueryRow("SELECT COUNT(*) FROM seat_reservations WHERE date=? AND train_class=? AND train_name=? AND car_number=? AND seat_row=? AND seat_column=?",
			date,
			trainClass,
			trainName,
			carNumber,
			seatRow,
			seatColumn).Scan(&result)
		s := TrainSeat{seatRow, seatColumn, seatClass, isSmokingSeat, false}
		if result == 1 {
			s.IsOccupied = true
		}
		seats = append(seats, s)

		// fmt.Fprintf(w, "%d,%d\n", distance, fare)
	}
	c := CarInformation{date, trainClass, trainName, carNumber, seats}
	resp, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	w.Write(resp)

	err = rows.Err()
	if err != nil {
		panic(err)
	}
}

func main() {
	// MySQL関連のお膳立て
	var err error

	host := os.Getenv("MYSQL_HOSTNAME")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("MYSQL_PORT")
	if port == "" {
		port = "3306"
	}
	_, err = strconv.Atoi(port)
	if err != nil {
		port = "3306"
	}
	user := os.Getenv("MYSQL_USER")
	if user == "" {
		user = "isutrain"
	}
	dbname := os.Getenv("MYSQL_DATABASE")
	if dbname == "" {
		dbname = "isutrain"
	}
	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		password = "isutrain"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user,
		password,
		host,
		port,
		dbname,
	)

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// HTTP
	http.HandleFunc("/api/train/search", trainSearchHandler)
	http.HandleFunc("/api/train/seats", trainSeatsHandler)

	http.ListenAndServe(":8000", nil)
}