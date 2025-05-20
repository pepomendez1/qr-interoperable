package utils

import (
	"database/sql"
	"fmt"
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

// --- EMVCo & QR Interoperable ---

// tag genera una entrada TLV con Tag, Longitud y Valor.
func tag(id, valor string) string {
	length := fmt.Sprintf("%02d", len(valor))
	return id + length + valor
}

// calcula el CRC16-CCITT (polinomio 0x1021) requerido por EMVCo
func calcularCRC16(data string) string {
	const poly = 0x1021
	crc := 0xFFFF

	for _, b := range []byte(data) {
		crc ^= int(b) << 8
		for i := 0; i < 8; i++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	crc &= 0xFFFF
	return strings.ToUpper(fmt.Sprintf("%04X", crc))
}

// GenerarPayloadEMVCo crea el string final codificable como QR con formato interoperable.
func GenerarPayloadEMVCo(qrID string, alias string, monto float64) string {
	dominioInvertido := "ar.com.wibond"

	// Subtag 26 (dominio y qrID)
	subtag26 := tag("00", dominioInvertido) + tag("01", qrID)
	tag26 := tag("26", subtag26)

	// Estructura base
	tags := ""
	tags += tag("52", "4722")                     // Rubro comercio (supermercado)
	tags += tag("53", "032")                      // Moneda ARS
	tags += tag("54", fmt.Sprintf("%.2f", monto)) // Monto
	tags += tag("58", "AR")                       // País
	tags += tag("59", "Disco")                    // Nombre
	tags += tag("60", "Cordoba")                  // Ciudad
	tags += tag("62", tag("01", qrID))            // ID de transacción
	tags += tag("98", "12345678")                 // ID ficticio de NewPay

	// Concatenación sin CRC final
	payloadSinCRC := tag("00", "01") + tag("01", "11") + tag26 + tags + "6304"

	crc := calcularCRC16(payloadSinCRC)
	payloadFinal := payloadSinCRC + crc

	return payloadFinal
}
