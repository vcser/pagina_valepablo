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
	writer.WriteAll(guests)
	if err != nil {
		fmt.Printf("Error al escribir en responses.csv: %v\n", err)
	}
}

// ... Los manejadores de `login`, `confirmation`, `text` y `success` son los mismos ...

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if _, ok := users[username]; !ok {
			newUser := User{Username: username, Password: password}
			users[username] = newUser
			saveNewUser(newUser)
			http.Redirect(w, r, "/confirmation", http.StatusSeeOther)
			return
		}

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

// Manejador para el formulario (con múltiples invitados).
func formHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Procesamos el formulario con múltiples invitados.
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error al procesar el formulario", http.StatusBadRequest)
			return
		}

		var guests [][]string

		// El formulario tiene campos como 'guests[0][fullname]', 'guests[1][fullname]', etc.
		// Iteramos sobre los índices para obtener cada invitado.
		for i := 0; ; i++ {
			fullname := r.FormValue(fmt.Sprintf("guests[%d][fullname]", i))
			email := r.FormValue(fmt.Sprintf("guests[%d][email]", i))
			phone := r.FormValue(fmt.Sprintf("guests[%d][phone]", i))
			isAdult := r.FormValue(fmt.Sprintf("guests[%d][isAdult]", i))
			allergies := r.FormValue(fmt.Sprintf("guests[%d][allergies]", i))
			song := r.FormValue(fmt.Sprintf("guests[%d][song]", i))

			// Si el nombre está vacío, significa que hemos llegado al final de los invitados.
			if fullname == "" {
				break
			}

			// Agregamos los datos del invitado al slice.
			guests = append(guests, []string{fullname, email, phone, isAdult, allergies, song})
		}

		// Guardamos todos los invitados en el archivo CSV.
		saveGuestData(guests)

		http.Redirect(w, r, "/success", http.StatusSeeOther)
		return
	}

	tmpl, _ := template.ParseFiles("templates/form.html")
	tmpl.Execute(w, nil)
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("templates/success.html")
	tmpl.Execute(w, "¡Formulario enviado con éxito!")
}

func main() {
	loadUsers()

	http.HandleFunc("/", loginHandler)
	http.HandleFunc("/confirmation", confirmationHandler)
	http.HandleFunc("/text", textHandler)
	http.HandleFunc("/form", formHandler)
	http.HandleFunc("/success", successHandler)

	fmt.Println("Servidor iniciado en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
