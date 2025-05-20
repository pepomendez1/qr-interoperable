package main

import (
	"fmt"
	"log"
	"net/http"

	"qrproject/handlers"
	"qrproject/utils"

	_ "modernc.org/sqlite"
)

func main() {
	db := utils.InitDB("../data/qr.db")
	handlers.SetDB(db)

	http.HandleFunc("/generar-qr", func(w http.ResponseWriter, r *http.Request) {
		handlers.GenerarQRHandler(db, w, r)
	})
	http.HandleFunc("/estado-pago/", func(w http.ResponseWriter, r *http.Request) {
		handlers.EstadoPagoHandler(db, w, r)
	})
	http.HandleFunc("/simular-pago/", func(w http.ResponseWriter, r *http.Request) {
		handlers.SimularPagoHandler(db, w, r)
	})
	http.HandleFunc("/todas-transacciones", func(w http.ResponseWriter, r *http.Request) {
		handlers.TodasTransaccionesHandler(db, w, r)
	})
	http.HandleFunc("/eliminar-transaccion/", func(w http.ResponseWriter, r *http.Request) {
		handlers.EliminarTransaccionHandler(db, w, r)
	})

	http.HandleFunc("/resolve-qr/", func(w http.ResponseWriter, r *http.Request) {
		handlers.ResolveQRHandler(db, w, r)
	})

	http.HandleFunc("/webhook/pago", func(w http.ResponseWriter, r *http.Request) {
		handlers.WebhookPagoHandler(db, w, r)
	})

	http.HandleFunc("/simular-pago/alias/", func(w http.ResponseWriter, r *http.Request) {
		handlers.SimularPagoPorQRIDHandler(db, w, r)
	})

	http.Handle("/cliente-scan.html", http.FileServer(http.Dir("../frontend")))

	http.HandleFunc("/", handlers.RootHandler)

	fmt.Println("Servidor corriendo en http://localhost:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))

}
