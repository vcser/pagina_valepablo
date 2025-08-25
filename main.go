package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
)

// saveAttendance guarda la asistencia del usuario ("no" si no asiste)
func saveAttendance(username, status string) {
	mu.Lock()
	defer mu.Unlock()
	var records [][]string
	file, err := os.OpenFile("attendance.csv", os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		reader := csv.NewReader(file)
		records, _ = reader.ReadAll()
		file.Close()
	}
	found := false
	for i, rec := range records {
		if len(rec) >= 2 && rec[0] == username {
			records[i][1] = status
			found = true
			break
		}
	}
	if !found {
		records = append(records, []string{username, status})
	}
	file, err = os.OpenFile("attendance.csv", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err == nil {
		writer := csv.NewWriter(file)
		writer.WriteAll(records)
		writer.Flush()
		file.Close()
	}
}

// getAttendance obtiene la asistencia del usuario ("no" si no asiste)
func getAttendance(username string) string {
	mu.Lock()
	defer mu.Unlock()
	file, err := os.Open("attendance.csv")
	if err != nil {
		return ""
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return ""
	}
	for _, rec := range records {
		if len(rec) >= 2 && rec[0] == username {
			return rec[1]
		}
	}
	return ""
}

// saveUserMessage guarda el mensaje de un usuario en messages.csv (sobrescribe si ya existe)
func saveUserMessage(username, message string) {
	mu.Lock()
	defer mu.Unlock()

	// Leer todos los mensajes existentes
	var records [][]string
	file, err := os.OpenFile("messages.csv", os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		reader := csv.NewReader(file)
		records, _ = reader.ReadAll()
		file.Close()
	}

	// Actualizar o agregar el mensaje del usuario
	found := false
	for i, rec := range records {
		if len(rec) >= 2 && rec[0] == username {
			records[i][1] = message
			found = true
			break
		}
	}
	if !found {
		records = append(records, []string{username, message})
	}

	// Escribir todos los mensajes de nuevo
	file, err = os.OpenFile("messages.csv", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err == nil {
		writer := csv.NewWriter(file)
		writer.WriteAll(records)
		writer.Flush()
		file.Close()
	}
}

// getUserMessage obtiene el mensaje guardado para un usuario
func getUserMessage(username string) string {
	mu.Lock()
	defer mu.Unlock()
	file, err := os.Open("messages.csv")
	if err != nil {
		return ""
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return ""
	}
	for _, rec := range records {
		if len(rec) >= 2 && rec[0] == username {
			return rec[1]
		}
	}
	return ""
}

// User representa la estructura de un usuario.
type User struct {
	Username string
	Password string
}

var mu sync.Mutex
var users = make(map[string]User)

// loadUsers carga los usuarios desde el archivo CSV.
func loadUsers() {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Open("users.csv")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Error al abrir users.csv: %v\n", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Error al leer users.csv: %v\n", err)
		return
	}

	for _, record := range records {
		if len(record) == 2 {
			users[record[0]] = User{Username: record[0], Password: record[1]}
		}
	}
}

// saveNewUser guarda un nuevo usuario en el archivo CSV.
func saveNewUser(user User) {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile("users.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Error al abrir users.csv para escritura: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write([]string{user.Username, user.Password})
	if err != nil {
		fmt.Printf("Error al escribir en users.csv: %v\n", err)
	}
}

// saveGuestData guarda los datos de cada invitado en el archivo CSV de respuestas.
// Now includes username as the first field for each guest
func saveGuestData(guests [][]string) {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile("responses.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Error al abrir responses.csv: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Escribimos cada invitado como una nueva fila.
	if err := writer.WriteAll(guests); err != nil {
		fmt.Printf("Error al escribir en responses.csv: %v\n", err)
	}
}

func isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value == "" {
		return false
	}
	_, exists := users[cookie.Value]
	return exists
}

// ... Los manejadores de `login`, `confirmation`, `text` y `success` son los mismos ...

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if _, ok := users[username]; !ok {
			// Usuario nuevo: crear y redirigir a /confirmation
			newUser := User{Username: username, Password: password}
			users[username] = newUser
			saveNewUser(newUser)
			http.Redirect(w, r, "/confirmation", http.StatusSeeOther)
			return
		}

		if users[username].Password == password {
			// Usuario existente: guardar cookie y redirigir según asistencia
			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: username,
				Path:  "/",
			})
			attendance := getAttendance(username)
			if attendance == "no" {
				http.Redirect(w, r, "/text", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/success", http.StatusSeeOther)
			}
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl, _ := template.ParseFiles("templates/login.html")
	tmpl.Execute(w, nil)
}

func confirmationHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		cookie, _ := r.Cookie("session")
		username := ""
		if cookie != nil && cookie.Value != "" {
			username = cookie.Value
		}
		if r.FormValue("confirm") == "si" {
			saveAttendance(username, "si")
			http.Redirect(w, r, "/form", http.StatusSeeOther)
			return
		}
		saveAttendance(username, "no")
		http.Redirect(w, r, "/text", http.StatusSeeOther)
		return
	}

	tmpl, _ := template.ParseFiles("templates/confirmation.html")
	tmpl.Execute(w, nil)
}

func textHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Obtener el usuario de la cookie de sesión
	cookie, err := r.Cookie("session")
	var username string
	if err == nil && cookie.Value != "" {
		username = cookie.Value
	} else {
		username = ""
	}

	if r.Method == http.MethodPost {
		// Guardar mensaje enviado y redirigir a success con tab=3
		message := r.FormValue("message")
		saveUserMessage(username, message)
		http.Redirect(w, r, "/success?tab=3", http.StatusSeeOther)
		return
	}

	// Si GET, mostrar formulario como antes (opcional, pero ya no se usará)
	userMessage := getUserMessage(username)
	data := struct {
		Message string
	}{
		Message: userMessage,
	}
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	tmpl.ExecuteTemplate(w, "text.html", data)
}

// Manejador para el formulario (con múltiples invitados).
func formHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	var username string
	if err == nil && cookie.Value != "" {
		username = cookie.Value
	} else {
		username = ""
	}

	// Eliminar modo edición: solo permitir agregar invitados nuevos

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error al procesar el formulario", http.StatusBadRequest)
			return
		}

		var guests [][]string
		for i := 0; ; i++ {
			fullname := r.FormValue(fmt.Sprintf("guests[%d][fullname]", i))
			email := r.FormValue(fmt.Sprintf("guests[%d][email]", i))
			phone := r.FormValue(fmt.Sprintf("guests[%d][phone]", i))
			isAdult := r.FormValue(fmt.Sprintf("guests[%d][isAdult]", i))
			allergies := r.FormValue(fmt.Sprintf("guests[%d][allergies]", i))
			song := r.FormValue(fmt.Sprintf("guests[%d][song]", i))
			if fullname == "" {
				break
			}
			guests = append(guests, []string{username, fullname, email, phone, isAdult, allergies, song})
		}
		saveGuestData(guests)
		http.Redirect(w, r, "/success", http.StatusSeeOther)
		return
	}

	// GET: no cargar invitados ni modo edición, solo renderizar formulario vacío
	tmpl, _ := template.ParseFiles("templates/form.html")
	tmpl.Execute(w, map[string]interface{}{})
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Obtener el usuario de la cookie de sesión
	cookie, err := r.Cookie("session")
	var username string
	if err == nil && cookie.Value != "" {
		username = cookie.Value
	} else {
		username = ""
	}

	// Leer responses.csv y filtrar los invitados de este usuario
	var userGuests [][]string
	file, err := os.Open("responses.csv")
	if err == nil {
		defer file.Close()
		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err == nil {
			for _, record := range records {
				if len(record) >= 2 && record[0] == username {
					userGuests = append(userGuests, record[1:]) // omit username
				}
			}
		}
	}

	// Leer mensaje del usuario
	userMessage := getUserMessage(username)

	// Leer tab de la query (por defecto 1)
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "1"
	}

	data := struct {
		Guests  [][]string
		Message string
		Tab     string
	}{
		Guests:  userGuests,
		Message: userMessage,
		Tab:     tab,
	}

	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	tmpl.ExecuteTemplate(w, "success.html", data)
}

func main() {
	loadUsers()

	// Servir archivos estáticos (imágenes, CSS, etc.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", loginHandler)
	http.HandleFunc("/confirmation", confirmationHandler)
	http.HandleFunc("/text", textHandler)
	http.HandleFunc("/form", formHandler)
	http.HandleFunc("/success", successHandler)

	fmt.Println("Servidor iniciado en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
