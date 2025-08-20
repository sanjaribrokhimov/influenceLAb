package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type BlogPost struct {
	ID            int      `json:"id"`
	Img           string   `json:"img"`
	Images        []string `json:"images"`
	Title         string   `json:"title"`
	TitleUz       string   `json:"title_uz"`
	TitleEn       string   `json:"title_en"`
	Description   string   `json:"description"`
	DescriptionUz string   `json:"description_uz"`
	DescriptionEn string   `json:"description_en"`
	Links         []string `json:"links"`
}

type FormRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
}

type Project struct {
	ID            int      `json:"id"`
	Img           string   `json:"img"`
	Images        []string `json:"images"`
	Title         string   `json:"title"`
	TitleUz       string   `json:"title_uz"`
	TitleEn       string   `json:"title_en"`
	Description   string   `json:"description"`
	DescriptionUz string   `json:"description_uz"`
	DescriptionEn string   `json:"description_en"`
	Links         []string `json:"links"`
}

// LED Screens entity
// price храним как float64, location текст
// images как JSON TEXT, img — первое изображение
// Совместимая схема и обработчики в стиле blog/projects

type LedItem struct {
	ID            int      `json:"id"`
	Img           string   `json:"img"`
	Images        []string `json:"images"`
	Title         string   `json:"title"`
	TitleUz       string   `json:"title_uz"`
	TitleEn       string   `json:"title_en"`
	Description   string   `json:"description"`
	DescriptionUz string   `json:"description_uz"`
	DescriptionEn string   `json:"description_en"`
	Location      string   `json:"location"`
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
	// LED Screens API
	http.HandleFunc("/api/led", withCORS(handleLed))
	http.HandleFunc("/api/led/", withCORS(handleLedByID))
	// Translation API
	http.HandleFunc("/api/translate", withCORS(handleTranslate))

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
		title_uz TEXT,
		title_en TEXT,
		description TEXT,
		description_uz TEXT,
		description_en TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}
	// Projects table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		img TEXT,
		title TEXT,
		title_uz TEXT,
		title_en TEXT,
		description TEXT,
		description_uz TEXT,
		description_en TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}
	// LED Screens table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS led (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		img TEXT,
		title TEXT,
		title_uz TEXT,
		title_en TEXT,
		description TEXT,
		description_uz TEXT,
		description_en TEXT,
		location TEXT,
		images TEXT
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
	if err := ensureColumn("blog", "title_uz", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("blog", "title_en", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("blog", "description_uz", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("blog", "description_en", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "images", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "links", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "title_uz", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "title_en", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "description_uz", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("projects", "description_en", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("led", "images", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("led", "title_uz", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("led", "title_en", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("led", "description_uz", "TEXT"); err != nil {
		log.Fatal(err)
	}
	if err := ensureColumn("led", "description_en", "TEXT"); err != nil {
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

// Функция перевода текста через Google Translate API
func translateText(text, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	// Определяем код языка для API
	var langCode string
	switch targetLang {
	case "uz":
		langCode = "uz"
	case "en":
		langCode = "en"
	default:
		return text, nil // Возвращаем оригинальный текст для неизвестных языков
	}

	// Запрос в Google Translate API (неофициальный)
	url := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=ru&tl=%s&dt=t&q=%s",
		langCode, url.QueryEscape(text))

	log.Printf("[translate] Requesting: %s", url)

	// Создаем клиент с заголовками
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[translate] Create request error: %v", err)
		return "", err
	}

	// Добавляем заголовки для имитации браузера
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// Убираем gzip из Accept-Encoding чтобы получить несжатый ответ
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[translate] HTTP error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[translate] Read body error: %v", err)
		return "", err
	}

	log.Printf("[translate] Response: %s", string(body))

	// Парсим JSON ответ от Google Translate
	var data []interface{}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&data); err != nil {
		log.Printf("[translate] JSON decode error: %v", err)
		return "", err
	}

	log.Printf("[translate] Parsed data: %+v", data)

	// Структура ответа: [translations, source_lang, target_lang, ...]
	if len(data) > 0 {
		if translations, ok := data[0].([]interface{}); ok && len(translations) > 0 {
			if translation, ok := translations[0].([]interface{}); ok && len(translation) > 0 {
				if translatedText, ok := translation[0].(string); ok {
					log.Printf("[translate] Success: %s -> %s", text, translatedText)
					return translatedText, nil
				}
			}
		}
	}

	log.Printf("[translate] No translation found, returning original")
	return text, nil // Возвращаем оригинальный текст в случае ошибки
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
		rows, err := db.Query("SELECT id, img, title, IFNULL(title_uz,''), IFNULL(title_en,''), description, IFNULL(description_uz,''), IFNULL(description_en,''), IFNULL(images,''), IFNULL(links,'') FROM blog ORDER BY id DESC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var posts []BlogPost
		for rows.Next() {
			var p BlogPost
			var imagesJSON, linksJSON string
			if err := rows.Scan(&p.ID, &p.Img, &p.Title, &p.TitleUz, &p.TitleEn, &p.Description, &p.DescriptionUz, &p.DescriptionEn, &imagesJSON, &linksJSON); err == nil {
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
			titleUz := r.FormValue("title_uz")
			titleEn := r.FormValue("title_en")
			desc := r.FormValue("description")
			descUz := r.FormValue("description_uz")
			descEn := r.FormValue("description_en")
			imagesJSON, _ := json.Marshal(images)
			linksJSON, _ := json.Marshal(links)
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			res, err := db.Exec("INSERT INTO blog (img, title, title_uz, title_en, description, description_uz, description_en, images, links) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", imgSingle, title, titleUz, titleEn, desc, descUz, descEn, string(imagesJSON), string(linksJSON))
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
		res, err := db.Exec("INSERT INTO blog (img, title, title_uz, title_en, description, description_uz, description_en, images, links) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", imgSingle, p.Title, p.TitleUz, p.TitleEn, p.Description, p.DescriptionUz, p.DescriptionEn, string(imagesJSON), string(linksJSON))
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
		err := db.QueryRow("SELECT id, img, title, IFNULL(title_uz,''), IFNULL(title_en,''), description, IFNULL(description_uz,''), IFNULL(description_en,''), IFNULL(images,''), IFNULL(links,'') FROM blog WHERE id = ?", id).Scan(&p.ID, &p.Img, &p.Title, &p.TitleUz, &p.TitleEn, &p.Description, &p.DescriptionUz, &p.DescriptionEn, &imagesJSON, &linksJSON)
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
			titleUz := r.FormValue("title_uz")
			titleEn := r.FormValue("title_en")
			desc := r.FormValue("description")
			descUz := r.FormValue("description_uz")
			descEn := r.FormValue("description_en")
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
			_, err := db.Exec("UPDATE blog SET img=?, title=?, title_uz=?, title_en=?, description=?, description_uz=?, description_en=?, images=?, links=? WHERE id=?", imgSingle, title, titleUz, titleEn, desc, descUz, descEn, string(imagesJSON), string(linksJSON), id)
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
		_, err := db.Exec("UPDATE blog SET img=?, title=?, title_uz=?, title_en=?, description=?, description_uz=?, description_en=?, images=?, links=? WHERE id=?", imgSingle, p.Title, p.TitleUz, p.TitleEn, p.Description, p.DescriptionUz, p.DescriptionEn, string(imagesJSON), string(linksJSON), id)
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
		rows, err := db.Query("SELECT id, img, title, IFNULL(title_uz,''), IFNULL(title_en,''), description, IFNULL(description_uz,''), IFNULL(description_en,''), IFNULL(images,''), IFNULL(links,'') FROM projects ORDER BY id DESC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var items []Project
		for rows.Next() {
			var p Project
			var imagesJSON, linksJSON string
			if err := rows.Scan(&p.ID, &p.Img, &p.Title, &p.TitleUz, &p.TitleEn, &p.Description, &p.DescriptionUz, &p.DescriptionEn, &imagesJSON, &linksJSON); err == nil {
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
			titleUz := r.FormValue("title_uz")
			titleEn := r.FormValue("title_en")
			desc := r.FormValue("description")
			descUz := r.FormValue("description_uz")
			descEn := r.FormValue("description_en")
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			imagesJSON, _ := json.Marshal(images)
			linksJSON, _ := json.Marshal(links)
			res, err := db.Exec("INSERT INTO projects (img, title, title_uz, title_en, description, description_uz, description_en, images, links) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", imgSingle, title, titleUz, titleEn, desc, descUz, descEn, string(imagesJSON), string(linksJSON))
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
		res, err := db.Exec("INSERT INTO projects (img, title, title_uz, title_en, description, description_uz, description_en, images, links) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", imgSingle, p.Title, p.TitleUz, p.TitleEn, p.Description, p.DescriptionUz, p.DescriptionEn, string(imagesJSON), string(linksJSON))
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
		err := db.QueryRow("SELECT id, img, title, IFNULL(title_uz,''), IFNULL(title_en,''), description, IFNULL(description_uz,''), IFNULL(description_en,''), IFNULL(images,''), IFNULL(links,'') FROM projects WHERE id = ?", id).Scan(&p.ID, &p.Img, &p.Title, &p.TitleUz, &p.TitleEn, &p.Description, &p.DescriptionUz, &p.DescriptionEn, &imagesJSON, &linksJSON)
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
			titleUz := r.FormValue("title_uz")
			titleEn := r.FormValue("title_en")
			desc := r.FormValue("description")
			descUz := r.FormValue("description_uz")
			descEn := r.FormValue("description_en")
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
			_, err := db.Exec("UPDATE projects SET img=?, title=?, title_uz=?, title_en=?, description=?, description_uz=?, description_en=?, images=?, links=? WHERE id=?", imgSingle, title, titleUz, titleEn, desc, descUz, descEn, string(imagesJSON), string(linksJSON), id)
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
		_, err := db.Exec("UPDATE projects SET img=?, title=?, title_uz=?, title_en=?, description=?, description_uz=?, description_en=?, images=?, links=? WHERE id=?", imgSingle, p.Title, p.TitleUz, p.TitleEn, p.Description, p.DescriptionUz, p.DescriptionEn, string(imagesJSON), string(linksJSON), id)
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

// --- LED CRUD ---
func handleLed(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, img, title, IFNULL(title_uz,''), IFNULL(title_en,''), description, IFNULL(description_uz,''), IFNULL(description_en,''), IFNULL(location,''), IFNULL(images,'') FROM led ORDER BY id DESC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var items []LedItem
		for rows.Next() {
			var it LedItem
			var imagesJSON string
			if err := rows.Scan(&it.ID, &it.Img, &it.Title, &it.TitleUz, &it.TitleEn, &it.Description, &it.DescriptionUz, &it.DescriptionEn, &it.Location, &imagesJSON); err == nil {
				if imagesJSON != "" {
					_ = json.Unmarshal([]byte(imagesJSON), &it.Images)
				}
				items = append(items, it)
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
			images = clampStrings(images, 10)
			title := r.FormValue("title")
			titleUz := r.FormValue("title_uz")
			titleEn := r.FormValue("title_en")
			desc := r.FormValue("description")
			descUz := r.FormValue("description_uz")
			descEn := r.FormValue("description_en")
			loc := r.FormValue("location")
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			imagesJSON, _ := json.Marshal(images)
			res, err := db.Exec("INSERT INTO led (img, title, title_uz, title_en, description, description_uz, description_en, location, images) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", imgSingle, title, titleUz, titleEn, desc, descUz, descEn, loc, string(imagesJSON))
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			id, _ := res.LastInsertId()
			it := LedItem{ID: int(id), Img: imgSingle, Images: images, Title: title, Description: desc, Location: loc}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(it)
			return
		}
		var it LedItem
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		it.Images = clampStrings(it.Images, 10)
		imgSingle := ""
		if len(it.Images) > 0 {
			imgSingle = it.Images[0]
		}
		imagesJSON, _ := json.Marshal(it.Images)
		res, err := db.Exec("INSERT INTO led (img, title, title_uz, title_en, description, description_uz, description_en, location, images) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", imgSingle, it.Title, it.TitleUz, it.TitleEn, it.Description, it.DescriptionUz, it.DescriptionEn, it.Location, string(imagesJSON))
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		it.ID = int(id)
		it.Img = imgSingle
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(it)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLedByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/led/")
	if id == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		var it LedItem
		var imagesJSON string
		err := db.QueryRow("SELECT id, img, title, IFNULL(title_uz,''), IFNULL(title_en,''), description, IFNULL(description_uz,''), IFNULL(description_en,''), IFNULL(location,''), IFNULL(images,'') FROM led WHERE id=?", id).Scan(&it.ID, &it.Img, &it.Title, &it.TitleUz, &it.TitleEn, &it.Description, &it.DescriptionUz, &it.DescriptionEn, &it.Location, &imagesJSON)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if imagesJSON != "" {
			_ = json.Unmarshal([]byte(imagesJSON), &it.Images)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(it)
	case http.MethodPost, http.MethodPut:
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}
			title := r.FormValue("title")
			titleUz := r.FormValue("title_uz")
			titleEn := r.FormValue("title_en")
			desc := r.FormValue("description")
			descUz := r.FormValue("description_uz")
			descEn := r.FormValue("description_en")
			loc := r.FormValue("location")
			var images []string
			if oldJSON := strings.TrimSpace(r.FormValue("imagesOld")); oldJSON != "" {
				_ = json.Unmarshal([]byte(oldJSON), &images)
			}
			if len(images) == 0 {
				var cur string
				_ = db.QueryRow("SELECT IFNULL(images,'') FROM led WHERE id=?", id).Scan(&cur)
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
			images = clampStrings(images, 10)
			imgSingle := ""
			if len(images) > 0 {
				imgSingle = images[0]
			}
			imagesJSON, _ := json.Marshal(images)
			_, err := db.Exec("UPDATE led SET img=?, title=?, title_uz=?, title_en=?, description=?, description_uz=?, description_en=?, location=?, images=? WHERE id=?", imgSingle, title, titleUz, titleEn, desc, descUz, descEn, loc, string(imagesJSON), id)
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		var it LedItem
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		it.Images = clampStrings(it.Images, 10)
		imgSingle := ""
		if len(it.Images) > 0 {
			imgSingle = it.Images[0]
		}
		imagesJSON, _ := json.Marshal(it.Images)
		_, err := db.Exec("UPDATE led SET img=?, title=?, title_uz=?, title_en=?, description=?, description_uz=?, description_en=?, location=?, images=? WHERE id=?", imgSingle, it.Title, it.TitleUz, it.TitleEn, it.Description, it.DescriptionUz, it.DescriptionEn, it.Location, string(imagesJSON), id)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	case http.MethodDelete:
		_, err := db.Exec("DELETE FROM led WHERE id=?", id)
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

// --- TRANSLATION API ---
type TranslateRequest struct {
	Text string `json:"text"`
	Lang string `json:"lang"` // "uz" или "en"
}

type TranslateResponse struct {
	Original     string            `json:"original"`
	Translated   string            `json:"translated,omitempty"`
	Translations map[string]string `json:"translations,omitempty"`
	Lang         string            `json:"lang"`
}

func handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	if req.Lang != "uz" && req.Lang != "en" && req.Lang != "all" {
		http.Error(w, "Lang must be 'uz', 'en' or 'all'", http.StatusBadRequest)
		return
	}

	var response TranslateResponse
	response.Original = req.Text
	response.Lang = req.Lang

	if req.Lang == "all" {
		// Переводим на все языки
		translations := make(map[string]string)

		// Перевод на UZ
		if uzTranslated, err := translateText(req.Text, "uz"); err == nil {
			translations["uz"] = uzTranslated
		}

		// Перевод на EN
		if enTranslated, err := translateText(req.Text, "en"); err == nil {
			translations["en"] = enTranslated
		}

		response.Translations = translations
	} else {
		// Перевод на один язык
		translated, err := translateText(req.Text, req.Lang)
		if err != nil {
			http.Error(w, "Translation failed", http.StatusInternalServerError)
			return
		}
		response.Translated = translated
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
