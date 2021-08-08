package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"html/template"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var templates *template.Template
var db *sql.DB

type Movie struct {
	ID          int
	Name        string
	Director    string
	Duration    string
	Description string
}

type Timings struct {
	ID        int
	MovieID   int
	Name      string
	Time      string
	Price     int
	Total     int
	Purchased int
}

type Tickets struct {
	ID       int
	TimingID int
	Num      int
}

func main() {
	db, err := sql.Open("sqlite3", "./movie.db")
	if err != nil {
		panic(err)
	}

	db.Exec(`CREATE TABLE users (
		"name" TEXT,
		"password" TEXT
	);`)

	statement, _ := db.Prepare("CREATE TABLE IF NOT EXISTS users (name TEXT, password TEXT)")
	statement.Exec()

	db.Exec(`CREATE TABLE movies (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"name" VARCHAR(20),
		"director" VARCHAR(20),
		"duration" VARCHAR(6), 
		"description" TEXT
	);`)

	db.Exec(`CREATE TABLE timings (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"movie_id" integer,
		"name" VARCHAR(20),
		"time" VARCHAR(6)
		"price" integer,
		"total" integer,
		"purchased" integer
	);`)

	db.Exec(`CREATE TABLE tickets (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"timing_id" integer,
		"num" integer
	);`)

	templates = template.Must(template.ParseGlob("assets/*.html"))

	r := mux.NewRouter()

	r.HandleFunc("/register", userRegistration).Methods("GET")
	r.HandleFunc("/login", userLogin).Methods("GET")
	r.HandleFunc("register", reg).Methods("POST")
	r.HandleFunc("/login", login).Methods("POST")

	r.HandleFunc("/dashboard", movieBooking).Methods("GET")
	r.HandleFunc("/add-movie", addMovie).Methods("POST")
	r.HandleFunc("/add-timings", addTiming).Methods("POST")
	r.HandleFunc("/purchase-tickets", purchaseTicket).Methods("POST")

	http.ListenAndServe(":8000", r)
}

/*func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}*/

func userRegistration(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "register.html", nil)
}

func reg(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.PostForm.Get("name")
	pwd := r.PostForm.Get("password")
	hashed, _ := hashPassword(pwd)

	//register user
	db.Exec(`
		INSERT INTO users(name,password) VALUES (?,?);`,
		name, hashed,
	)

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func userLogin(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "login.html", nil)
}

func login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	/*
		name := r.Form.Get("username")
		pwd := r.Form.Get("password")
		hashed, _ := hashPassword(pwd)
		statement, _ := db.Prepare("SELECT * FROM users WHERE name=? AND password=?")
		/*res, err := statement.Exec(name, hashed)
		if err != nil {
			log.Fatalln(err.Error())
		}*/
	//if res.RowsAffe
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

/*func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}*/

func movieBooking(w http.ResponseWriter, r *http.Request) {
	data := struct {
		SearchResults   []Movie
		AllMovies       []Movie
		AllMovieTimings []Timings
		Price           int
	}{
		Price:           0,
		SearchResults:   []Movie{},
		AllMovies:       []Movie{},   //load movies here
		AllMovieTimings: []Timings{}, //load timings here
	}

	allMovies, _ := db.Query(`SELECT * FROM movies;`)
	defer allMovies.Close()
	for allMovies.Next() {
		mov := Movie{}
		allMovies.Scan(&mov.ID, &mov.Name, &mov.Director, &mov.Duration, &mov.Description)
		data.AllMovies = append(data.AllMovies, mov)
	}

	allMovieTimings, _ := db.Query(`SELECT * FROM timings;`)
	defer allMovieTimings.Close()
	for allMovieTimings.Next() {
		tim := Timings{}
		allMovieTimings.Scan(&tim.ID, &tim.MovieID, &tim.Name, &tim.Time, &tim.Price, &tim.Total, &tim.Purchased)
		data.AllMovieTimings = append(data.AllMovieTimings, tim)
	}

	q, ok := r.URL.Query()["q"]
	if ok {
		fmt.Println(q[0])
		qs, _ := db.Query(`SELECT * FROM movies WHERE name=?`, q[0])
		defer qs.Close()

		for qs.Next() {
			mov := Movie{}
			qs.Scan(&mov.ID, &mov.Name, &mov.Director, &mov.Duration, &mov.Description)
			data.SearchResults = append(data.SearchResults, mov)
		}
	}

	templates.ExecuteTemplate(w, "index.html", data)
}

func addMovie(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	movie := Movie{}
	movie.Name = r.PostForm.Get("name")
	movie.Director = r.PostForm.Get("director")
	movie.Duration = r.PostForm.Get("duration")
	movie.Description = r.PostForm.Get("description")

	db.Exec(
		`INSERT INTO movies(name, director, duration, description) VALUES (?,?,?,?);`,
		movie.Name, movie.Director, movie.Duration, movie.Description,
	)
	//	fmt.Println(movie)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func addTiming(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	timing := Timings{}
	idName := strings.Split(r.PostForm.Get("id"), "/")
	timing.MovieID, _ = strconv.Atoi(idName[0])
	timing.Name = idName[1]
	timing.Time = r.PostForm.Get("time")
	timing.Price, _ = strconv.Atoi(r.PostForm.Get("price"))
	timing.Total, _ = strconv.Atoi(r.PostForm.Get("total"))

	if timing.Price <= 0 || timing.Total < 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	db.Exec(`
	INSERT INTO timings(movie_id, name,time, price, total, purchased) VALUES (?,?,?,?,?,?);`,
		timing.MovieID, timing.Name, timing.Time, timing.Price, timing.Total, 0,
	)

	//fmt.Printf(timing)
	http.Redirect(w, r, "/dashboard", http.StatusFound)

}

func purchaseTicket(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ticket := Tickets{}
	ticket.TimingID, _ = strconv.Atoi(r.PostForm.Get("id"))
	ticket.Num, _ = strconv.Atoi(r.PostForm.Get("num"))

	total, purchased := 0, 0
	res, _ := db.Query(`SELECT * FROM timings where id =?`, ticket.TimingID)
	defer res.Close()
	for res.Next() {
		tim := Timings{}
		res.Scan(&tim.ID, &tim.MovieID, &tim.Name, &tim.Time, &tim.Price, &tim.Total, &tim.Purchased)
		//price = tim.Price
		total = tim.Total
		purchased = tim.Purchased
		break
	}

	if ticket.Num > total-purchased {
		w.Write([]byte(fmt.Sprintf(`
		<h1>Error</h1>
		<p>only %d tickets left</p>
		`, total-purchased)))
		return
	}

	db.Exec(`
		INSERT INTO tickets(timing_id,num) VALUES (?,?);
		UPDATE tmings SET purchased = ? WHERE id = ?`,
		ticket.TimingID, ticket.Num, ticket.Num+purchased, ticket.TimingID,
	)

	//fmt.Println(ticket)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}
