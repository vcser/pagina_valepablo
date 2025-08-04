package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
)

// User representa la estructura de un usuario.
type User struct {
	Username string
	Password string
}

// Mutex para evitar que varios goroutines escriban en el archivo al mismo tiempo.
var mu sync.Mutex

// "Base de datos" en memoria para el login, se carga desde el CSV.
var users = make(map[string]User)

// loadUsers carga los usuarios desde el archivo CSV.
func loadUsers() {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Open("users.csv")
	if err != nil {
		// Si el archivo no existe, lo creamos.
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

// saveFormData guarda los datos del formulario en el archivo CSV de respuestas.
func saveFormData(name, email string) {
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

	err = writer.Write([]string{name, email})
	if err != nil {
		fmt.Printf("Error al escribir en responses.csv: %v\n", err)
	}
}

// Manejador para la página de inicio (login/registro).
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if _, ok := users[username]; !ok {
			// Si el usuario no existe, lo registramos y guardamos en el CSV.
			newUser := User{Username: username, Password: password}
			users[username] = newUser
			saveNewUser(newUser)
			fmt.Printf("Nuevo usuario registrado: %s\n", username)
			http.Redirect(w, r, "/confirmation", http.StatusSeeOther)
			return
		}

		// Si existe, validamos la contraseña.
		if users[username].Password == password {
			http.Redirect(w, r, "/confirmation", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl, _ := template.ParseFiles("templates/login.html")
	tmpl.Execute(w, nil)
}

// ... (El resto de los manejadores, confirmationHandler, textHandler, confirmationHandler, formHandler, successHandler, permanecen iguales, excepto formHandler)

// Manejador para la página del formulario (cuando la respuesta fue "si").
func formHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		email := r.FormValue("email")

		// Guardamos los datos del formulario en el archivo CSV de respuestas.
		saveFormData(name, email)

		http.Redirect(w, r, "/success", http.StatusSeeOther)
		return
	}

	tmpl, _ := template.ParseFiles("templates/form.html")
	tmpl.Execute(w, nil)
}

func main() {
	// Cargamos los usuarios existentes al inicio de la aplicación.
	loadUsers()

	// Definimos las rutas.
	http.HandleFunc("/", loginHandler)
	http.HandleFunc("/confirmation", confirmationHandler)
	http.HandleFunc("/text", textHandler)
	http.HandleFunc("/form", formHandler)
	http.HandleFunc("/success", successHandler)

	fmt.Println("Servidor iniciado en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// ... (Funciones auxiliares como confirmationHandler, textHandler, successHandler)
func confirmationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.FormValue("confirm") == "si" {
			http.Redirect(w, r, "/form", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/text", http.StatusSeeOther)
		return
	}

	tmpl, _ := template.ParseFiles("templates/confirmation.html")
	tmpl.Execute(w, nil)
}

func textHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("templates/text.html")
	tmpl.Execute(w, "Has seleccionado 'No'. ¡Gracias por tu visita!")
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("templates/success.html")
	tmpl.Execute(w, "¡Formulario enviado con éxito!")
}
