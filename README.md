# 🧾 QR Interoperable - Sistema de Pagos

Este proyecto es una implementación básica de un sistema de pagos mediante QR interoperable, orientado a comercios que deseen recibir pagos desde distintas billeteras virtuales.
Incluye backend en Go y frontend en HTML/JS con diseño inspirado en el estilo visual de Wibond.

---

## 📁 Estructura del Proyecto
qr-project/
├── backend/
│ ├── handlers/ → Lógica de endpoints (generación de QR, pagos, historial)
│ ├── utils/ → Funciones comunes: CORS, auth, parseo
│ └── main.go → Configura rutas y arranca el servidor
├── data/
│ └── qr.db → Base de datos SQLite (ignorada por Git)
├── frontend/
│ ├── index.html → Panel del comercio
│ └── cliente.html → Panel del cliente
├── go.mod → Configuración del módulo Go
├── go.sum → Sumas de dependencias (auto-generado)
└── README.md → Este archivo

---

## 🚀 Funcionalidades Implementadas

- ✅ Generación de QR con payload `{alias, monto}`
- ✅ Simulación de pago por parte del cliente
- ✅ Estado de la transacción en tiempo real
- ✅ Expiración automática de pagos pasados 5 minutos
- ✅ Eliminación manual de transacciones
- ✅ Visualización de últimas 5 transacciones
- ✅ Autenticación simple vía Bearer Token
- ✅ Modularización por paquetes Go (`handlers` / `utils`)
- ✅ Persistencia real con SQLite (`qr.db`)

---

## ▶ Cómo correr el backend

Desde la raíz del proyecto:

```bash
go run backend/main.go
```

---

(Asegurate de tener la BD creada en data/qr.db)
CREATE TABLE transacciones (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  alias TEXT,
  monto REAL,
  payload TEXT,
  estado TEXT,
  creado_en TEXT,
  cliente_alias TEXT
);

---

🧪 Frontends
👨‍💼 Comercio → frontend/index.html
Genera QR

Simula pagos

Consulta estado

Elimina transacciones

🙋 Cliente → frontend/cliente.html
Ingresa ID del QR

Simula lectura y pago

Recibe feedback visual

Utiliza token para autorización

---

Token de Autorizacion
Authorization: Bearer mipassword123

--- 

## 📱 Escanear y pagar desde el celular

### 1. Generar un QR
- Desde `index.html`, generar un QR como comercio.

### 2. Iniciar el servidor
Asegurarse de tener el archivo `cliente-scan.html` servido desde el backend:

```go
http.Handle("/cliente-scan.html", http.FileServer(http.Dir("./frontend")))
```
Y correr:
```
go run main.go
```
3. Exponer el backend con ngrok
```
ngrok http 8080
```

Se obtiene una URL pública HTTPS como:
https://abc123.ngrok-free.app

4. Acceder desde el celular
En el navegador del celular, abrir:
https://abc123.ngrok-free.app/cliente-scan.html
Se activa la cámara

Se escanea el QR generado

Se extrae el qr_id

Se resuelve y muestra el comercio y monto

Botón “Pagar ahora” simula el webhook y marca el QR como pagado ✅

---

🛣 Roadmap
🔄 Integración real con Newpay

🔐 Validación antifraude con KOIN

📩 Notificaciones vía SNS

💰 Lógica de liquidación a CVU

🧪 Tests unitarios

🗃 Persistencia avanzada


