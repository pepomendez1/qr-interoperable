package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"qrproject/utils"

	"github.com/skip2/go-qrcode"
)

var db *sql.DB

func SetDB(database *sql.DB) {
	db = database
}

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

func GenerarQRHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	utils.SetCORSHeaders(w)

	if !utils.AuthorizeRequest(w, r) {
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
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

	id64, _ := res.LastInsertId()
	resp := QRResponse{
		ID:       int(id64),
		QRBase64: base64QR,
		Payload:  payload,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func EstadoPagoHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	utils.SetCORSHeaders(w)

	id, err := utils.ParseIDFromURL(r.URL.Path, "/estado-pago/")
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var estado string
	err = db.QueryRow("SELECT estado FROM transacciones WHERE id = ?", id).Scan(&estado)
	if err != nil {
		http.Error(w, "Transacción no encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"estado": estado})
}

func SimularPagoHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	utils.SetCORSHeaders(w)
	if !utils.AuthorizeRequest(w, r) {
		return
	}
	id, err := utils.ParseIDFromURL(r.URL.Path, "/simular-pago/")
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	utils.ActualizarEstado(id)

	var estado string
	err = db.QueryRow("SELECT estado FROM transacciones WHERE id = ?", id).Scan(&estado)
	if err != nil || estado != "pendiente" {
		http.Error(w, "Transacción no puede ser pagada", http.StatusForbidden)
		return
	}

	var body struct {
		ClienteAlias string `json:"cliente_alias"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE transacciones SET estado = 'pagado', cliente_alias = ? WHERE id = ?", body.ClienteAlias, id)
	if err != nil {
		http.Error(w, "Error al actualizar estado", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func TodasTransaccionesHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	utils.SetCORSHeaders(w)
	if !utils.AuthorizeRequest(w, r) {
		return
	}

	rows, err := db.Query(`
		SELECT id, alias, monto, payload, estado, creado_en, cliente_alias
		FROM transacciones ORDER BY id ASC`)
	if err != nil {
		log.Println("ERROR CONSULTA DB:", err)
		http.Error(w, "Error al consultar transacciones", http.StatusInternalServerError)
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
		if cliente.Valid {
			t.ClienteAlias = cliente.String
		}
		lista = append(lista, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lista)
}

func EliminarTransaccionHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	utils.SetCORSHeaders(w)
	if !utils.AuthorizeRequest(w, r) {
		return
	}

	id, err := utils.ParseIDFromURL(r.URL.Path, "/eliminar-transaccion/")
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM transacciones WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Error al eliminar", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Servidor QR Interoperable activo 🚀")
}
