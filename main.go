package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

// Article структура статьи
type Article struct {
	Id, Category_id         uint16
	Title, Anons, Full_text string
	Image                   sql.NullString
}

// Category структура категории
type Category struct {
	ID   int
	Name string
}

var store = sessions.NewCookieStore([]byte("super-secret-key"))

var tpl *template.Template

func init() {
	tpl = template.Must(template.ParseGlob("templates/*.html"))
}

// Получение текущего пользователя
func getUser(r *http.Request) string {
	session, _ := store.Get(r, "session")
	if username, ok := session.Values["username"].(string); ok {
		return username
	}
	return ""
}

// Главная страница
func index(w http.ResponseWriter, r *http.Request) {
	sortCol := r.URL.Query().Get("sort")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var res *sql.Rows
	if sortCol == "" {
		res, err = db.Query(`SELECT id, title, anons, full_text, image, category_id FROM articles`)
	} else {
		res, err = db.Query(`SELECT id, title, anons, full_text, image, category_id FROM articles WHERE category_id = ?`, sortCol)
	}
	if err != nil {
		panic(err)
	}
	defer res.Close()

	var posts []Article
	for res.Next() {
		var p Article
		err = res.Scan(&p.Id, &p.Title, &p.Anons, &p.Full_text, &p.Image, &p.Category_id)
		if err != nil {
			panic(err)
		}

		if !p.Image.Valid {
			p.Image.String = ""
		}

		posts = append(posts, p)
	}

	var categories []Category
	rows, _ := db.Query("SELECT id, name FROM categories")
	defer rows.Close()
	for rows.Next() {
		var c Category
		rows.Scan(&c.ID, &c.Name)
		categories = append(categories, c)
	}

	tpl.ExecuteTemplate(w, "index", map[string]interface{}{
		"User":       getUser(r),
		"Posts":      posts,
		"Categories": categories,
	})

}

// Показ статьи
func show_post(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var post Article
	err = db.QueryRow(`SELECT id, title, anons, full_text, image, category_id FROM articles WHERE id = ?`, vars["id"]).
		Scan(&post.Id, &post.Title, &post.Anons, &post.Full_text, &post.Image, &post.Category_id)
	if err != nil {
		http.Error(w, "Статья не найдена", http.StatusNotFound)
		return
	}

	if !post.Image.Valid {
		post.Image.String = ""
	}

	tpl.ExecuteTemplate(w, "show", map[string]interface{}{
		"User": getUser(r),
		"Post": post,
	})
}

// Создание статьи
func create(w http.ResponseWriter, r *http.Request) {
	if getUser(r) == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			panic(err)
		}
		categories = append(categories, c)
	}

	tpl.ExecuteTemplate(w, "create", map[string]interface{}{
		"User":       getUser(r),
		"Categories": categories,
	})
}

// Сохранение статьи
func save_article(w http.ResponseWriter, r *http.Request) {
	if getUser(r) == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.ParseMultipartForm(10 << 20)
	title := r.FormValue("title")
	anons := r.FormValue("anons")
	full_text := r.FormValue("full_text")
	category_id := r.FormValue("category_id")

	imagePath := ""
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		if _, err := os.Stat("uploads"); os.IsNotExist(err) {
			os.Mkdir("uploads", os.ModePerm)
		}
		dst, err := os.Create("uploads/" + handler.Filename)
		if err != nil {
			panic(err)
		}
		defer dst.Close()
		io.Copy(dst, file)
		imagePath = "uploads/" + handler.Filename
	}

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO articles (title, anons, full_text, image, category_id) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(title, anons, full_text, imagePath, category_id)
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Регистрация
func register(w http.ResponseWriter, r *http.Request) {

	tpl.ExecuteTemplate(w, "register", nil)
}

func registerPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		fmt.Fprintf(w, "Все поля обязательны!")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Ошибка хеширования", http.StatusInternalServerError)
		return
	}

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, email, string(hash))
	if err != nil {
		fmt.Fprintf(w, "Ошибка: %v", err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Логин
func login(w http.ResponseWriter, r *http.Request) {

	tpl.ExecuteTemplate(w, "login", nil)
}

func loginPost(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var id int
	var username, passwordHash string
	err = db.QueryRow("SELECT id, username, password_hash FROM users WHERE email = ?", email).
		Scan(&id, &username, &passwordHash)
	if err != nil {
		fmt.Fprintf(w, "Пользователь не найден")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		fmt.Fprintf(w, "Неверный пароль")
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["user_id"] = id
	session.Values["username"] = username
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Выход
func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Маршрутен
func handleFunc() {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/", index).Methods("GET")
	rtr.HandleFunc("/create", create).Methods("GET")
	rtr.HandleFunc("/save_article", save_article).Methods("POST")
	rtr.HandleFunc("/post/{id:[0-9]+}", show_post).Methods("GET")
	rtr.HandleFunc("/register", register).Methods("GET")
	rtr.HandleFunc("/register", registerPost).Methods("POST")
	rtr.HandleFunc("/login", login).Methods("GET")
	rtr.HandleFunc("/login", loginPost).Methods("POST")
	rtr.HandleFunc("/logout", logout).Methods("GET")

	// Статическе файлен
	rtr.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	rtr.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	http.Handle("/", rtr)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	handleFunc()
}
