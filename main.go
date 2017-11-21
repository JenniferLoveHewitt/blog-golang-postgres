package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	_ "github.com/lib/pq"

	"./models"
)

var (
	db *sql.DB
)

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

const (
	DB_USER     = "user"
	DB_PASSWORD = "pswrd"
	DB_NAME     = "dbname"
)

func main() {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		DB_USER, DB_PASSWORD, DB_NAME)

	db, _ = sql.Open("postgres", dbinfo)

	defer db.Close()

	router := mux.NewRouter().StrictSlash(true)

	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/articles", indexHandler).Methods("GET")

	router.HandleFunc("/articles/new", newHandler).Methods("GET")
	router.HandleFunc("/create", postCreateHandler).Methods("POST")

	router.HandleFunc("/articles/{id}", showHandler).Methods("GET")

	router.HandleFunc("/edit/{id}", editHandler).Methods("GET")
	router.HandleFunc("/update", postUpdateHandler).Methods("POST")

	router.HandleFunc("/delete/{id}", deleteHandler)

	router.HandleFunc("/users", usersHandler).Methods("GET")

	router.HandleFunc("/users/new", registrationHandler).Methods("GET")
	router.HandleFunc("/createuser", createUserPostHandler).Methods("POST")

	router.HandleFunc("/users/{id}", userInfoHandler).Methods("GET")

	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/auth", postLoginHandler).Methods("POST")

	router.HandleFunc("/logout", logoutHandler).Methods("POST")

	router.HandleFunc("/account", accountHandler).Methods("GET")

	log.Fatal(http.ListenAndServe(":3001", router))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	rows, err := db.Query("SELECT * FROM articles")

	defer rows.Close()

	articles := make([]*models.Article, 0)

	for rows.Next() {
		article := new(models.Article)

		err := rows.Scan(&article.Id,
			&article.Category,
			&article.Title,
			&article.Subtitle,
			&article.Content,
			&article.User_id,
			&article.Created)

		PanicOnErr(err)

		db.QueryRow("SELECT login FROM userinfo WHERE uid = $1", article.User_id).Scan(&article.Login)

		articles = append(articles, article)
	}

	t.ExecuteTemplate(w, "index", articles)
}

func showHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/show.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	params := mux.Vars(r)
	id := params["id"]

	row := db.QueryRow("SELECT * FROM articles WHERE uid = $1", id)
	article := new(models.Article)

	row.Scan(&article.Id,
		&article.Category,
		&article.Title,
		&article.Subtitle,
		&article.Content,
		&article.User_id,
		&article.Created)

	db.QueryRow("SELECT login FROM userinfo WHERE uid = $1", &article.User_id).Scan(&article.Login)

	t.ExecuteTemplate(w, "show", article)
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/new.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	login, _ := getUserInfo(r)

	if login == "" {
		loginHandler(w, r)
		return
	}

	t.ExecuteTemplate(w, "new", nil)
}

func postCreateHandler(w http.ResponseWriter, r *http.Request) {
	category := r.FormValue("category")
	title := r.FormValue("title")
	subtitle := r.FormValue("subtitle")
	content := r.FormValue("content")
	login, _ := getUserInfo(r)
	uid := 0

	db.QueryRow("SELECT uid FROM userinfo WHERE login = $1", login).Scan(&uid)
	db.QueryRow("INSERT INTO articles (category, title, subtitle, content, user_uid, created) VALUES ($1, $2, $3, $4, $5, $6)",
		category,
		title,
		subtitle,
		content,
		uid,
		time.Now())

	http.Redirect(w, r, "/", 302)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	uid := 0
	user_id := 0
	login, _ := getUserInfo(r)

	db.QueryRow("SELECT uid FROM userinfo WHERE login = $1", login).Scan(&uid)
	db.QueryRow("SELECT user_uid FROM articles WHERE uid = $1", id).Scan(&user_id)

	if uid == 0 || user_id != uid {
		http.Redirect(w, r, "/", 302)
		return
	}

	_, err := db.Exec("DELETE FROM articles WHERE uid = $1", id)

	PanicOnErr(err)

	http.Redirect(w, r, "/", 302)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/edit.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	params := mux.Vars(r)
	id := params["id"]

	row := db.QueryRow("SELECT * FROM articles WHERE uid = $1", id)
	article := new(models.Article)
	row.Scan(&article.Id,
		&article.Category,
		&article.Title,
		&article.Subtitle,
		&article.Content,
		&article.Created)

	t.ExecuteTemplate(w, "edit", article)
}

func postUpdateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	category := r.FormValue("category")
	title := r.FormValue("title")
	subtitle := r.FormValue("subtitle")
	content := r.FormValue("content")

	_, err := db.Exec("UPDATE articles SET category = $1, title = $2, subtitle = $3, content = $4, created = $5 WHERE uid = $6",
		category,
		title,
		subtitle,
		content,
		time.Now(),
		id)

	PanicOnErr(err)

	http.Redirect(w, r, "/", 302)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/users.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	rows, err := db.Query("SELECT * FROM userinfo")

	defer rows.Close()

	users := make([]*models.UserInfo, 0)

	for rows.Next() {
		user := new(models.UserInfo)

		err := rows.Scan(&user.Id,
			&user.Login,
			&user.Email,
			&user.Password,
			&user.Created,
			&user.Role)

		PanicOnErr(err)

		users = append(users, user)
	}

	t.ExecuteTemplate(w, "users", users)
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/registration.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	t.ExecuteTemplate(w, "reg", nil)
}

func createUserPostHandler(w http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confPassword := r.FormValue("confpassword")

	errs := make([]string, 0)
	if confPassword != password {
		errs = append(errs, "Пароли не совпадают")
	}
	if len(password) < 8 {
		errs = append(errs, "Пароль слишком короткий")
	}
	if len(login) < 6 {
		errs = append(errs, "Логин слишком короткий")
	}

	if len(errs) != 0 {
		t, err := template.ParseFiles("templates/registration.html",
			"templates/header.html",
			"templates/footer.html")

		PanicOnErr(err)

		t.ExecuteTemplate(w, "reg", errs)

		return
	}

	db.QueryRow("INSERT INTO userinfo (login, email, password, created, role) VALUES ($1, $2, $3, $4, $5)",
		login,
		email,
		password,
		time.Now(),
		"User")

	http.Redirect(w, r, "/", 302)
}

func userInfoHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/user.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	params := mux.Vars(r)
	id := params["id"]

	rows, err := db.Query("SELECT * FROM articles WHERE user_uid = $1", id)
	PanicOnErr(err)

	articles := make([]*models.Article, 0)
	for rows.Next() {
		article := new(models.Article)

		err := rows.Scan(&article.Id,
			&article.Category,
			&article.Title,
			&article.Subtitle,
			&article.Content,
			&article.User_id,
			&article.Created)

		PanicOnErr(err)

		db.QueryRow("SELECT login FROM userinfo WHERE uid = $1", article.User_id).Scan(&article.Login)
		articles = append(articles, article)
	}

	t.ExecuteTemplate(w, "user", articles)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/login.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	t.ExecuteTemplate(w, "login", nil)
}

func postLoginHandler(w http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	password := r.FormValue("password")

	row := db.QueryRow("SELECT * FROM userinfo WHERE login = $1", login)

	user := new(models.UserInfo)
	row.Scan(&user.Id,
		&user.Login,
		&user.Email,
		&user.Password,
		&user.Created,
		&user.Role)

	errs := make([]string, 0)
	if password != user.Password {
		errs = append(errs, "Неправильный логин или пароль1")
	} else if row == nil {
		errs = append(errs, "Неправильный логин или пароль")
	}

	if len(errs) != 0 {
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html",
			"templates/footer.html")
		PanicOnErr(err)
		t.ExecuteTemplate(w, "login", errs)

		return
	}

	setSession(user.Login, user.Role, w)
	http.Redirect(w, r, "/account", 302)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/", 302)
}

func accountHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/account.html",
		"templates/header.html",
		"templates/footer.html")

	PanicOnErr(err)

	login, _ := getUserInfo(r)

	if login == "" {
		login = "you arent authorized"
	}

	t.ExecuteTemplate(w, "account", login)
}

func setSession(login string, role string, w http.ResponseWriter) {
	value := map[string]string{
		"login": login,
		"role": role,
	}

	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}

		http.SetCookie(w, cookie)
	}
}

func clearSession(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

func getUserInfo(r *http.Request) (login string, role string) {
	if cookie, err := r.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)

		if err = cookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			login = cookieValue["login"]
			role = cookieValue["role"]
		}
	}

	return login, role
}

func PanicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}