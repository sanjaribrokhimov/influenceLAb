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
	ID          int      `json:"id"`
	Img         string   `json:"img"`
	Images      []string `json:"images"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Links       []string `json:"links"`
}

type FormRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
}

type Project struct {
	ID          int      `json:"id"`
	Img         string   `json:"img"`
	Images      []string `json:"images"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Links       []string `json:"links"`
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
	// Ensure new columns exist
	if err := ensureColumn("blog", "images", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("blog", "links", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "images", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "links", "TEXT"); err != nil {
		log.Fatal(err)
	}
}

func ensureColumn(table string, column string, columnType string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	present := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err == nil {
			if name == column {
				present = true
				break
			}
		}
	}
	if present {
		return nil
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType))
	return err
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

func clampStrings(values []string, max int) []string {
	if len(values) > max {
		return values[:max]
	}
	return values
}

func parseLinksFromForm(r *http.Request) []string {
	links := []string{}
	if r.MultipartForm != nil {
		if arr, ok := r.MultipartForm.Value["links"]; ok {
			for _, v := range arr {
				v = strings.TrimSpace(v)
				if v != "" {
					links = append(links, v)
				}
			}
		}
		// Также поддержим link1..link5
		for i := 1; i <= 5; i++ {
			v := strings.TrimSpace(r.FormValue(fmt.Sprintf("link%d", i)))
			if v != "" {
				links = append(links, v)
			}
		}
	}
	return clampStrings(uniqueStrings(links), 5)
}

func uniqueStrings(arr []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, v := range arr {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
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
		rows, err := db.Query("SELECT id, img, title, description, IFNULL(images,''), IFNULL(links,'') FROM blog ORDER BY id DESC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var posts []BlogPost
		for rows.Next() {
			var p BlogPost
			var imagesJSON, linksJSON string
			if err := rows.Scan(&p.ID, &p.Img, &p.Title, &p.Description, &imagesJSON, &linksJSON); err == nil {
				if imagesJSON != "" {
					_ = json.Unmarshal([]byte(imagesJSON), &p.Images)
				}
				if linksJSON != "" {
					_ = json.Unmarshal([]byte(linksJSON), &p.Links)
				}
				posts = append(posts, p)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	case http.MethodPost:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			// Collect images
			var images []string
			if files, ok := r.MultipartForm.File["imgs"]; ok {
				for i, fh := range files {
					if i >= 10 {
						break
					}
					f, err := fh.Open()
					if err != nil {
						continue
					}
					path, err := saveUploadedFile(f, fh)
					f.Close()
					if err == nil {
						images = append(images, path)
					}
				}
			}
			if len(images) == 0 {
				if file, handler, err := r.FormFile("img"); err == nil {
					defer file.Close()
					if path, err := saveUploadedFile(file, handler); err == nil {
						images = append(images, path)
					}
				}
			}
			images = clampStrings(images, 10)
			links := parseLinksFromForm(r)
			title := r.FormValue("title")
			desc := r.FormValue("description")
			imagesJSON, _ := json.Marshal(images)
			linksJSON, _ := json.Marshal(links)
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			res, err := db.Exec("INSERT INTO blog (img, title, description, images, links) VALUES (?, ?, ?, ?, ?)", imgSingle, title, desc, string(imagesJSON), string(linksJSON))
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			id, _ := res.LastInsertId()
			p := BlogPost{ID: int(id), Img: imgSingle, Images: images, Title: title, Description: desc, Links: links}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
		// JSON body
		var p BlogPost
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		p.Images = clampStrings(p.Images, 10)
		p.Links = clampStrings(uniqueStrings(p.Links), 5)
		imgSingle := ""
		if len(p.Images) > 0 {
			imgSingle = p.Images[0]
		}
		imagesJSON, _ := json.Marshal(p.Images)
		linksJSON, _ := json.Marshal(p.Links)
		res, err := db.Exec("INSERT INTO blog (img, title, description, images, links) VALUES (?, ?, ?, ?, ?)", imgSingle, p.Title, p.Description, string(imagesJSON), string(linksJSON))
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		p.ID = int(id)
		p.Img = imgSingle
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
		var imagesJSON, linksJSON string
		err := db.QueryRow("SELECT id, img, title, description, IFNULL(images,''), IFNULL(links,'') FROM blog WHERE id = ?", id).Scan(&p.ID, &p.Img, &p.Title, &p.Description, &imagesJSON, &linksJSON)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if imagesJSON != "" {
			_ = json.Unmarshal([]byte(imagesJSON), &p.Images)
		}
		if linksJSON != "" {
			_ = json.Unmarshal([]byte(linksJSON), &p.Links)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	case http.MethodPost, http.MethodPut:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			title := r.FormValue("title")
			desc := r.FormValue("description")
			// Start with old images if provided
			var images []string
			if oldJSON := strings.TrimSpace(r.FormValue("imagesOld")); oldJSON != "" {
				_ = json.Unmarshal([]byte(oldJSON), &images)
			}
			// Or fallback to current DB value
			if len(images) == 0 {
				var cur string
				_ = db.QueryRow("SELECT IFNULL(images,'') FROM blog WHERE id=?", id).Scan(&cur)
				if cur != "" {
					_ = json.Unmarshal([]byte(cur), &images)
				}
			}
			// Add new files
			if files, ok := r.MultipartForm.File["imgs"]; ok {
				for i, fh := range files {
					if i >= 10 {
						break
					}
					f, err := fh.Open()
					if err != nil {
						continue
					}
					path, err := saveUploadedFile(f, fh)
					f.Close()
					if err == nil {
						images = append(images, path)
					}
				}
			}
			// Optional single replacement
			if file, handler, err := r.FormFile("img"); err == nil {
				defer file.Close()
				if path, err := saveUploadedFile(file, handler); err == nil {
					images = append([]string{path}, images...)
				}
			}
			images = clampStrings(images, 10)
			links := parseLinksFromForm(r)
			imagesJSON, _ := json.Marshal(images)
			linksJSON, _ := json.Marshal(links)
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			_, err := db.Exec("UPDATE blog SET img=?, title=?, description=?, images=?, links=? WHERE id=?", imgSingle, title, desc, string(imagesJSON), string(linksJSON), id)
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		// JSON
		var p BlogPost
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		p.Images = clampStrings(p.Images, 10)
		p.Links = clampStrings(uniqueStrings(p.Links), 5)
		imgSingle := ""
		if len(p.Images) > 0 {
			imgSingle = p.Images[0]
		}
		imagesJSON, _ := json.Marshal(p.Images)
		linksJSON, _ := json.Marshal(p.Links)
		_, err := db.Exec("UPDATE blog SET img=?, title=?, description=?, images=?, links=? WHERE id=?", imgSingle, p.Title, p.Description, string(imagesJSON), string(linksJSON), id)
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
		rows, err := db.Query("SELECT id, img, title, description, IFNULL(images,''), IFNULL(links,'') FROM projects ORDER BY id DESC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var items []Project
		for rows.Next() {
			var p Project
			var imagesJSON, linksJSON string
			if err := rows.Scan(&p.ID, &p.Img, &p.Title, &p.Description, &imagesJSON, &linksJSON); err == nil {
				if imagesJSON != "" {
					_ = json.Unmarshal([]byte(imagesJSON), &p.Images)
				}
				if linksJSON != "" {
					_ = json.Unmarshal([]byte(linksJSON), &p.Links)
				}
				items = append(items, p)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	case http.MethodPost:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			var images []string
			if files, ok := r.MultipartForm.File["imgs"]; ok {
				for i, fh := range files {
					if i >= 10 {
						break
					}
					f, err := fh.Open()
					if err != nil {
						continue
					}
					path, err := saveUploadedFile(f, fh)
					f.Close()
					if err == nil {
						images = append(images, path)
					}
				}
			}
			if len(images) == 0 {
				if file, handler, err := r.FormFile("img"); err == nil {
					defer file.Close()
					if path, err := saveUploadedFile(file, handler); err == nil {
						images = append(images, path)
					}
				}
			}
			images = clampStrings(images, 10)
			links := parseLinksFromForm(r)
			title := r.FormValue("title")
			desc := r.FormValue("description")
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			imagesJSON, _ := json.Marshal(images)
			linksJSON, _ := json.Marshal(links)
			res, err := db.Exec("INSERT INTO projects (img, title, description, images, links) VALUES (?, ?, ?, ?, ?)", imgSingle, title, desc, string(imagesJSON), string(linksJSON))
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			id, _ := res.LastInsertId()
			p := Project{ID: int(id), Img: imgSingle, Images: images, Title: title, Description: desc, Links: links}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
		var p Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		p.Images = clampStrings(p.Images, 10)
		p.Links = clampStrings(uniqueStrings(p.Links), 5)
		imgSingle := ""
		if len(p.Images) > 0 {
			imgSingle = p.Images[0]
		}
		imagesJSON, _ := json.Marshal(p.Images)
		linksJSON, _ := json.Marshal(p.Links)
		res, err := db.Exec("INSERT INTO projects (img, title, description, images, links) VALUES (?, ?, ?, ?, ?)", imgSingle, p.Title, p.Description, string(imagesJSON), string(linksJSON))
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		p.ID = int(id)
		p.Img = imgSingle
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
		var imagesJSON, linksJSON string
		err := db.QueryRow("SELECT id, img, title, description, IFNULL(images,''), IFNULL(links,'') FROM projects WHERE id = ?", id).Scan(&p.ID, &p.Img, &p.Title, &p.Description, &imagesJSON, &linksJSON)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if imagesJSON != "" {
			_ = json.Unmarshal([]byte(imagesJSON), &p.Images)
		}
		if linksJSON != "" {
			_ = json.Unmarshal([]byte(linksJSON), &p.Links)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	case http.MethodPost, http.MethodPut:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			title := r.FormValue("title")
			desc := r.FormValue("description")
			var images []string
			if oldJSON := strings.TrimSpace(r.FormValue("imagesOld")); oldJSON != "" {
				_ = json.Unmarshal([]byte(oldJSON), &images)
			}
			if len(images) == 0 {
				var cur string
				_ = db.QueryRow("SELECT IFNULL(images,'') FROM projects WHERE id=?", id).Scan(&cur)
				if cur != "" {
					_ = json.Unmarshal([]byte(cur), &images)
				}
			}
			if files, ok := r.MultipartForm.File["imgs"]; ok {
				for i, fh := range files {
					if i >= 10 {
						break
					}
					f, err := fh.Open()
					if err != nil {
						continue
					}
					path, err := saveUploadedFile(f, fh)
					f.Close()
					if err == nil {
						images = append(images, path)
					}
				}
			}
			if file, handler, err := r.FormFile("img"); err == nil {
				defer file.Close()
				if path, err := saveUploadedFile(file, handler); err == nil {
					images = append([]string{path}, images...)
				}
			}
			images = clampStrings(images, 10)
			links := parseLinksFromForm(r)
			imagesJSON, _ := json.Marshal(images)
			linksJSON, _ := json.Marshal(links)
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			_, err := db.Exec("UPDATE projects SET img=?, title=?, description=?, images=?, links=? WHERE id=?", imgSingle, title, desc, string(imagesJSON), string(linksJSON), id)
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
		p.Images = clampStrings(p.Images, 10)
		p.Links = clampStrings(uniqueStrings(p.Links), 5)
		imgSingle := ""
		if len(p.Images) > 0 {
			imgSingle = p.Images[0]
		}
		imagesJSON, _ := json.Marshal(p.Images)
		linksJSON, _ := json.Marshal(p.Links)
		_, err := db.Exec("UPDATE projects SET img=?, title=?, description=?, images=?, links=? WHERE id=?", imgSingle, p.Title, p.Description, string(imagesJSON), string(linksJSON), id)
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
