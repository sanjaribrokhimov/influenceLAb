package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type BlogPost struct {
	ID          int    `json:"id"`
	Img         string `json:"img"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type FormRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
}

type Project struct {
	ID          int    `json:"id"`
	Img         string `json:"img"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

var db *sql.DB

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h(w, r)
	}
}

func main() {
	// Загружаем переменные окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found or error loading it:", err)
	}

	var err error
	db, err = sql.Open("sqlite3", "file:influence.db?cache=shared&_fk=1")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	initDB()

	http.HandleFunc("/api/form", withCORS(handleForm))
	http.HandleFunc("/api/blog", withCORS(handleBlog))
	http.HandleFunc("/api/blog/", withCORS(handleBlogByID))
	// Projects API
	http.HandleFunc("/api/projects", withCORS(handleProjects))
	http.HandleFunc("/api/projects/", withCORS(handleProjectByID))

	// Static files and HTML pages
	rootDir := ".."
	fileServer := http.FileServer(http.Dir(rootDir))
	log.Println("Serving static files from:", rootDir)

	// Custom root handler: '/' -> index.html, '/about' -> about.html, fallback to static
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Do not handle API here
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(rootDir, "index.html"))
			return
		}
		// If no extension, try .html (e.g., /about -> /about.html)
		base := filepath.Base(r.URL.Path)
		if !strings.Contains(base, ".") {
			candidate := filepath.Join(rootDir, strings.TrimPrefix(r.URL.Path, "/")+".html")
			if _, err := os.Stat(candidate); err == nil {
				http.ServeFile(w, r, candidate)
				return
			}
		}
		// Fallback to static server (assets like /img/*, /components/*, etc.)
		fileServer.ServeHTTP(w, r)
	})

	log.Println("Server started on :9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

func initDB() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS blog (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		img TEXT,
		title TEXT,
		description TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}
	// Projects table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		img TEXT,
		title TEXT,
		description TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}
}

func saveUploadedFile(file multipart.File, handler *multipart.FileHeader) (string, error) {
	dir := "img/uploads"
	// Файлы сохраняем в корень проекта (на уровень выше api)
	fsDir := filepath.Join("..", dir)
	os.MkdirAll(fsDir, 0755)
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	fsPath := filepath.Join(fsDir, filename)
	f, err := os.Create(fsPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, file)
	if err != nil {
		return "", err
	}
	// Публичный URL относительно корня сайта
	urlPath := "/" + filepath.ToSlash(filepath.Join(dir, filename))
	return urlPath, nil
}

// --- FORM HANDLER ---
func handleForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req FormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// Отправка в Telegram
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		log.Printf("[form] Telegram env not set, skipping send. Name: %s Phone: %s", req.Name, req.Phone)
		http.Error(w, "Telegram config missing", http.StatusInternalServerError)
		return
	}
	msg := fmt.Sprintf("Новая заявка!\nИмя: %s\nТелефон: %s\nОписание: %s", req.Name, req.Phone, req.Description)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := strings.NewReader(fmt.Sprintf("chat_id=%s&text=%s", chatID, msg))
	resp, err := http.Post(url, "application/x-www-form-urlencoded", payload)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Failed to send to Telegram", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// --- BLOG CRUD ---
func handleBlog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, img, title, description FROM blog ORDER BY id DESC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var posts []BlogPost
		for rows.Next() {
			var p BlogPost
			if err := rows.Scan(&p.ID, &p.Img, &p.Title, &p.Description); err == nil {
				posts = append(posts, p)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	case http.MethodPost:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			err := r.ParseMultipartForm(10 << 20)
			if err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			file, handler, err := r.FormFile("img")
			if err != nil {
				http.Error(w, "Image required", http.StatusBadRequest)
				return
			}
			defer file.Close()
			imgPath, err := saveUploadedFile(file, handler)
			if err != nil {
				http.Error(w, "Failed to save image", http.StatusInternalServerError)
				return
			}
			title := r.FormValue("title")
			desc := r.FormValue("description")
			res, err := db.Exec("INSERT INTO blog (img, title, description) VALUES (?, ?, ?)", imgPath, title, desc)
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			id, _ := res.LastInsertId()
			p := BlogPost{ID: int(id), Img: imgPath, Title: title, Description: desc}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
		var p BlogPost
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		res, err := db.Exec("INSERT INTO blog (img, title, description) VALUES (?, ?, ?)", p.Img, p.Title, p.Description)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		p.ID = int(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleBlogByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/blog/")
	if id == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		var p BlogPost
		err := db.QueryRow("SELECT id, img, title, description FROM blog WHERE id = ?", id).Scan(&p.ID, &p.Img, &p.Title, &p.Description)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	case http.MethodPost, http.MethodPut:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			err := r.ParseMultipartForm(10 << 20)
			if err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			title := r.FormValue("title")
			desc := r.FormValue("description")
			imgPath := r.FormValue("imgOld")
			file, handler, err := r.FormFile("img")
			if err == nil {
				defer file.Close()
				imgPath, err = saveUploadedFile(file, handler)
				if err != nil {
					http.Error(w, "Failed to save image", http.StatusInternalServerError)
					return
				}
			}
			_, err = db.Exec("UPDATE blog SET img=?, title=?, description=? WHERE id=?", imgPath, title, desc, id)
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		var p BlogPost
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		_, err := db.Exec("UPDATE blog SET img=?, title=?, description=? WHERE id=?", p.Img, p.Title, p.Description, id)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	case http.MethodDelete:
		_, err := db.Exec("DELETE FROM blog WHERE id=?", id)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- PROJECTS CRUD ---
func handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, img, title, description FROM projects ORDER BY id DESC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var items []Project
		for rows.Next() {
			var p Project
			if err := rows.Scan(&p.ID, &p.Img, &p.Title, &p.Description); err == nil {
				items = append(items, p)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	case http.MethodPost:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			err := r.ParseMultipartForm(10 << 20)
			if err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			file, handler, err := r.FormFile("img")
			if err != nil {
				http.Error(w, "Image required", http.StatusBadRequest)
				return
			}
			defer file.Close()
			imgPath, err := saveUploadedFile(file, handler)
			if err != nil {
				http.Error(w, "Failed to save image", http.StatusInternalServerError)
				return
			}
			title := r.FormValue("title")
			desc := r.FormValue("description")
			res, err := db.Exec("INSERT INTO projects (img, title, description) VALUES (?, ?, ?)", imgPath, title, desc)
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			id, _ := res.LastInsertId()
			p := Project{ID: int(id), Img: imgPath, Title: title, Description: desc}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
		var p Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		res, err := db.Exec("INSERT INTO projects (img, title, description) VALUES (?, ?, ?)", p.Img, p.Title, p.Description)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		p.ID = int(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProjectByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if id == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		var p Project
		err := db.QueryRow("SELECT id, img, title, description FROM projects WHERE id = ?", id).Scan(&p.ID, &p.Img, &p.Title, &p.Description)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	case http.MethodPost, http.MethodPut:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			err := r.ParseMultipartForm(10 << 20)
			if err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			title := r.FormValue("title")
			desc := r.FormValue("description")
			imgPath := r.FormValue("imgOld")
			file, handler, err := r.FormFile("img")
			if err == nil {
				defer file.Close()
				imgPath, err = saveUploadedFile(file, handler)
				if err != nil {
					http.Error(w, "Failed to save image", http.StatusInternalServerError)
					return
				}
			}
			_, err = db.Exec("UPDATE projects SET img=?, title=?, description=? WHERE id=?", imgPath, title, desc, id)
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		var p Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		_, err := db.Exec("UPDATE projects SET img=?, title=?, description=? WHERE id=?", p.Img, p.Title, p.Description, id)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	case http.MethodDelete:
		_, err := db.Exec("DELETE FROM projects WHERE id=?", id)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
