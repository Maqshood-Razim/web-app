package main

import (
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	port = ":8080"
	// user = "admin"
	// pass = "123"
)

var (
	templates = template.Must(template.ParseFiles("client/login.html", "client/home.html", "client/signup.html", "client/admin.html", "client/adminhome.html", "client/create.html"))
	store     = sessions.NewCookieStore([]byte("super-secret-key"))
	db        *gorm.DB
)

type User struct {
	ID       uint   `gorm:"primarykey;not null;"`
	Username string `gorm:"type:varchar(50);not null;unique"`
	Password string `gorm:"type:varchar(50);not null"`
	Gmail    string `gorm:"type:varchar(50);not null"`
	IsAdmin  bool   `gorm:"type:boolean;not null;defualt:false"`
}

func admin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		var user User
		result := db.Where("username = ? AND password = ?", username, password).First(&user)
		if result.Error != nil {
			templates.ExecuteTemplate(w, "admin.html", map[string]string{"Error": "Invalid username or password"})
			return
		}

		session, _ := store.Get(r, "session_id")
		session.Values["authenticated"] = true
		session.Values["isAdmin"] = user.IsAdmin
		session.Save(r, w)

		if user.IsAdmin {
			http.Redirect(w, r, "/adminhome", http.StatusSeeOther)
		}

		templates.ExecuteTemplate(w, "admin.html", map[string]string{
			"Error": "You are not admin",
		})

		return

	}

	templates.ExecuteTemplate(w, "admin.html", nil)
}

func adminhome(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session_id")

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var users []User
	result := db.Find(&users)
	if result.Error != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	templates.ExecuteTemplate(w, "adminhome.html", map[string]interface{}{
		"Users": users,
	})
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		result := db.Delete(&User{}, id)
		if result.Error != nil {
			http.Error(w, "Error deleting user", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/adminhome", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func editUser(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	session, _ := store.Get(r, "session_id")

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	w.Header().Set("no-store", "cache-control , must-revalidate")
	w.Header().Set("pragma", "no-cache")

	var user User
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("gmail")

		result := db.Model(&user).Where("id = ?", id).Updates(User{Username: username, Password: password, Gmail: email})
		if result.Error != nil {
			http.Error(w, "User already exists", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/adminhome", http.StatusSeeOther)
		return
	}

	result := db.First(&user, id)
	if result.Error != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	t, err := template.ParseFiles("client/edit.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}

	t.Execute(w, user)
}

func search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	var users []User
	db.Where("username LIKE ? OR ID LIKE  ?", "%"+query+"%", "%"+query+"%").Find(&users)
	templates.ExecuteTemplate(w, "adminhome.html", map[string]interface{}{
		"Users": users,
	})
}

func create(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("gmail")

		session, _ := store.Get(r, "session_id")

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, "/adminhome", http.StatusSeeOther)
			return
		}

		user := User{Username: username, Password: password, Gmail: email}
		result := db.Create(&user)
		if result.Error != nil {
			templates.ExecuteTemplate(w, "create.html", map[string]string{"Error": "Username already exists"})
			return
		}

		w.Header().Set("control-cache", "no-store,must-revalidate")
		w.Header().Set("pragma", "no-cache")

		http.Redirect(w, r, "/adminhome", http.StatusSeeOther)

	}

	templates.ExecuteTemplate(w, "create.html", nil)
}

func signup(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		gmail := r.FormValue("gmail")

		if !Password(password) {
			templates.ExecuteTemplate(w, "signup.html", map[string]string{"Error": "Password must be at least 7 characters and numbers"})
			return
		}

		user := User{Username: username, Password: password, Gmail: gmail}
		result := db.Create(&user)
		if result.Error != nil {
			templates.ExecuteTemplate(w, "signup.html", map[string]string{"Error": "Username already exists"})
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	templates.ExecuteTemplate(w, "signup.html", nil)
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("username")

		var user User
		result := db.Where("(username = ? or gmail = ? ) AND password = ?", username, email, password).First(&user)
		if result.Error != nil {
			templates.ExecuteTemplate(w, "login.html", map[string]string{"Error": "Invalid username or password"})
			return
		}

		session, _ := store.Get(r, "session_id")
		session.Values["authenticated"] = true
		session.Save(r, w)
		http.Redirect(w, r, "/home", http.StatusSeeOther)

		return
	}
	templates.ExecuteTemplate(w, "login.html", nil)
}

func home(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session_id")

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	templates.ExecuteTemplate(w, "home.html", nil)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session_id")

	session.Values["authenticated"] = false
	session.Save(r, w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func Password(password string) bool {
	Number := regexp.MustCompile(`[0-9]`).MatchString(password)
	Length := len(password) >= 7
	return Number && Length
}

func main() {
	var err error
	dsn := "root:razeem19@tcp(localhost:3306)/db3?charset=utf8mb4&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	log.Println("Connection successful")
	db.AutoMigrate(&User{})

	mux := mux.NewRouter()
	mux.HandleFunc("/", home)
	mux.HandleFunc("/admin", admin).Methods(http.MethodGet, http.MethodPost)
	mux.HandleFunc("/adminhome", adminhome).Methods(http.MethodGet)
	mux.HandleFunc("/search", search).Methods(http.MethodGet)
	mux.HandleFunc("/home", home).Methods(http.MethodGet)
	mux.HandleFunc("/login", login).Methods(http.MethodGet, http.MethodPost)
	mux.HandleFunc("/logout", logout).Methods(http.MethodPost, http.MethodGet)
	mux.HandleFunc("/signup", signup).Methods(http.MethodGet, http.MethodPost)
	mux.HandleFunc("/create", create).Methods(http.MethodGet, http.MethodPost)
	mux.HandleFunc("/delete/{id:[0-9]+}", deleteUser).Methods(http.MethodGet, http.MethodPost)
	mux.HandleFunc("/edit/{id:[0-9]+}", editUser).Methods(http.MethodGet, http.MethodPost)

	mux.PathPrefix("/client/").Handler(http.StripPrefix("/client/", http.FileServer(http.Dir("client"))))

	log.Println("Server started on port", port)
	log.Fatal(http.ListenAndServe(port, mux))

	
}
