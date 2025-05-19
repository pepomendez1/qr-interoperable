package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	_ "modernc.org/sqlite"
)

type QRRequest struct {
	Monto float64 `json:"monto"`
	Alias string  `json:"alias"`
}

type QRResponse struct {
	ID       int    `json:"id"`
	QRBase64 string `json:"qr_base64"`
	Payload  string `json:"payload"`
}

type Transaccion struct {
	ID           int       `json:"ID"`
	Alias        string    `json:"Alias"`
	Monto        float64   `json:"Monto"`
	Payload      string    `json:"Payload"`
	Estado       string    `json:"Estado"`
	CreadoEn     time.Time `json:"CreadoEn"`
	ClienteAlias string    `json:"ClienteAlias"`
}

var db *sql.DB

func generarQRHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	if !checkAuth(r) {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var req QRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
		return
	}

	payload := fmt.Sprintf("ALIAS:%s|MONTO:%.2f|REF:QR123", req.Alias, req.Monto)
	png, err := qrcode.Encode(payload, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Error generando QR", http.StatusInternalServerError)
		return
	}

	base64QR := base64.StdEncoding.EncodeToString(png)
	res, err := db.Exec(`
		INSERT INTO transacciones (alias, monto, payload, estado, creado_en)
		VALUES (?, ?, ?, ?, ?)`,
		req.Alias, req.Monto, payload, "pendiente", time.Now().Format(time.RFC3339),
	)
	if err != nil {
		http.Error(w, "Error al guardar en base de datos", http.StatusInternalServerError)
		return
	}

	id64, err := res.LastInsertId()
	if err != nil {
		http.Error(w, "No se pudo obtener el ID", http.StatusInternalServerError)
		return
	}

	resp := QRResponse{
		ID:       int(id64),
		QRBase64: base64QR,
		Payload:  payload,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func estadoPagoHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/estado-pago/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var estado string
	err = db.QueryRow("SELECT estado FROM transacciones WHERE id = ?", id).Scan(&estado)
	if err == sql.ErrNoRows {
		http.Error(w, "Transacción no encontrada", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Error al consultar estado", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"estado": estado})
}

func simularPagoHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if !checkAuth(r) {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/simular-pago/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	actualizarEstado(id)

	var estado string
	err = db.QueryRow("SELECT estado FROM transacciones WHERE id = ?", id).Scan(&estado)
	if err == sql.ErrNoRows {
		http.Error(w, "Transacción no encontrada", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Error al verificar estado", http.StatusInternalServerError)
		return
	}
	if estado != "pendiente" {
		http.Error(w, "La transacción ya no puede ser pagada", http.StatusForbidden)
		return
	}

	var body struct {
		ClienteAlias string `json:"cliente_alias"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`UPDATE transacciones SET estado = 'pagado', cliente_alias = ? WHERE id = ?`, body.ClienteAlias, id)
	if err != nil {
		http.Error(w, "Error al actualizar estado", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func todasTransaccionesHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if !checkAuth(r) {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`
		SELECT id, alias, monto, payload, estado, creado_en, cliente_alias
		FROM transacciones ORDER BY id ASC`)
	if err != nil {
		http.Error(w, "Error al consultar transacciones", http.StatusInternalServerError)
		http.Error(w, "Error al consultar transacciones: "+err.Error(), http.StatusInternalServerError)

		return
	}
	defer rows.Close()

	var lista []Transaccion
	for rows.Next() {
		var t Transaccion
		var creadoStr string
		var cliente sql.NullString

		err := rows.Scan(&t.ID, &t.Alias, &t.Monto, &t.Payload, &t.Estado, &creadoStr, &cliente)
		if err != nil {
			log.Println("Error escaneando fila:", err)
			continue
		}
		t.CreadoEn, _ = time.Parse(time.RFC3339, creadoStr)
		t.ClienteAlias = ""
		if cliente.Valid {
			t.ClienteAlias = cliente.String
		}
		lista = append(lista, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lista)
}

func eliminarTransaccionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	token := r.Header.Get("Authorization")
	if token != "Bearer mipassword123" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/eliminar-transaccion/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	result, err := db.Exec("DELETE FROM transacciones WHERE id = ?", id)
	if err != nil {
		log.Println("Error al eliminar transacción:", err)
		http.Error(w, "Error al eliminar", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "No se encontró la transacción para eliminar", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	fmt.Fprintln(w, "Servidor QR Interoperable activo 🚀")
}

func actualizarEstado(id int) {
	var creadoEnStr, estado string
	err := db.QueryRow("SELECT creado_en, estado FROM transacciones WHERE id = ?", id).Scan(&creadoEnStr, &estado)
	if err != nil || estado != "pendiente" {
		return
	}
	creadoEn, err := time.Parse(time.RFC3339, creadoEnStr)
	if err != nil {
		return
	}
	if time.Since(creadoEn) > 5*time.Minute {
		db.Exec("UPDATE transacciones SET estado = 'expirado' WHERE id = ?", id)
	}
}

// --- helpers ---

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func checkAuth(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer mipassword123"
}

func main() {
	var err error
	db, err = sql.Open("sqlite", "../data/qr.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transacciones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			alias TEXT,
			monto REAL,
			payload TEXT,
			estado TEXT,
			creado_en TEXT
		)`)
	if err != nil {
		log.Fatal("Error creando tabla:", err)
	}

	_, err = db.Exec(`ALTER TABLE transacciones ADD COLUMN cliente_alias TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		log.Fatal("Error agregando columna cliente_alias:", err)
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/generar-qr", generarQRHandler)
	http.HandleFunc("/estado-pago/", estadoPagoHandler)
	http.HandleFunc("/simular-pago/", simularPagoHandler)
	http.HandleFunc("/todas-transacciones", todasTransaccionesHandler)
	http.HandleFunc("/eliminar-transaccion/", eliminarTransaccionHandler)

	fmt.Println("Servidor corriendo en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
