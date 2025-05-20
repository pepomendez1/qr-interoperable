package utils

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(path string) *sql.DB {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal("No se pudo abrir la base de datos:", err)
	}
	DB = database
	return DB
}

func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func AuthorizeRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return false
	}
	token := r.Header.Get("Authorization")
	if token != "Bearer mipassword123" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return false
	}
	return true
}

func ActualizarEstado(id int) {
	var creadoEnStr, estado string

	err := DB.QueryRow("SELECT creado_en, estado FROM transacciones WHERE id = ?", id).Scan(&creadoEnStr, &estado)
	if err != nil || estado != "pendiente" {
		return
	}

	creadoEn, err := time.Parse(time.RFC3339, creadoEnStr)
	if err != nil {
		return
	}

	if time.Since(creadoEn) > 5*time.Minute {
		_, err := DB.Exec("UPDATE transacciones SET estado = 'expirado' WHERE id = ?", id)
		if err != nil {
			log.Println("Error al actualizar a expirado:", err)
		}
	}
}

func ParseIDFromURL(path, prefix string) (int, error) {
	idStr := strings.TrimPrefix(path, prefix)
	return strconv.Atoi(idStr)
}
