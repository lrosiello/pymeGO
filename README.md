# pymeGO

# 🛍️ PymeGo Backend - SaaS Catalog & WhatsApp Ordering API

API backend robusta desarrollada en **Go** utilizando **Clean Architecture**, diseñada para la gestión de catálogos comerciales, control de inventario con variantes y stock atómico, autenticación JWT y un sistema integrado de pedidos con redirección directa a WhatsApp.

---

## 🚀 Stack Tecnológico

* **Lenguaje:** Go (v1.26+)
* **Framework Web:** Gin
* **Base de Datos & SQL:** PostgreSQL con `pgxpool` (v5) (SQL explícito, sin ORMs pesados)
* **Autenticación:** JWT (`golang-jwt/jwt/v5`) y encriptación de contraseñas con `bcrypt`
* **Validaciones:** `go-playground/validator/v10`
* **Arquitectura:** Clean Architecture (Domain, Usecase, Infrastructure, Delivery/HTTP)

---

## 📂 Estructura del Proyecto

```text
catalog-go/
├── cmd/
│   └── api/
│       └── main.go                  # Punto de entrada, inyección de dependencias y graceful shutdown
├── internal/
│   ├── domain/                      # Entidades puras de negocio e interfaces de repositorios
│   ├── usecase/                     # Lógica de aplicación y reglas de negocio transaccionales
│   ├── infrastructure/              
│   │   ├── config/                  # Carga de variables de entorno (.env)
│   │   └── persistence/postgres/    # Implementación de repositorios con pgxpool y TxManager
│   └── delivery/                    
│       └── http/                    # Capa HTTP (Gin)
│           ├── handler/             # Controladores (Auth, Products, Orders)
│           ├── middleware/          # JWT y control de roles (Admin/Staff)
│           └── dto/                 # Objetos de transferencia y validadores
├── pkg/                             # Utilidades agnósticas (Logger/slog, JWT, WhatsApp Builder)
├── migrations/                      # Scripts DDL de PostgreSQL
└── .env.example



🛠️ Base de Datos y Modelo de Datos
El sistema maneja un esquema relacional optimizado en PostgreSQL:

users: Gestión de usuarios, credenciales y roles (admin, staff, customer).

categories: Jerarquía opcional de categorías de productos.

products: Información genérica del escaparate (nombre, descripción, imágenes).

product_variants: Control atómico de stock por variante (talles, colores), SKU único y precios diferenciados (B2C y B2B mayorista).

order_logs: Registro de pedidos con snapshot de ítems en JSONB y cálculo automático de totales.

🔌 Endpoints Principales
Auth
POST /api/v1/auth/register — Registro de administrador inicial y emisión de JWT.

POST /api/v1/auth/login — Inicio de sesión de usuarios.

GET /api/v1/me — Datos del usuario autenticado (Protegido).

Catálogo y Productos
GET /categories / POST /categories — Gestión de categorías.

GET /products / POST /products — Listado y alta atómica de productos con variantes.

GET /products/:id / PUT /products/:id / DELETE /products/:id — Operaciones sobre productos.

POST /products/:id/variants — Agregar variantes a un producto existente.

PUT /variants/:id / DELETE /variants/:id — Actualizar o dar de baja variantes.

PATCH /variants/:id/stock — Ajuste atómico de stock por delta (ej: {"delta": -3}).

Pedidos & WhatsApp
POST /orders — Checkout público del cliente (descuenta stock de forma atómica, valida precios server-side y genera el enlace de WhatsApp).

GET /orders / GET /orders/:id — Listado y detalle de pedidos (Admin/Staff).

PATCH /orders/:id/status — Actualización de estados (pending, confirmed, shipped, cancelled con devolución automática de stock).

⚙️ Configuración y Ejecución Local
Clonar el repositorio:

Bash
git clone [https://github.com/lrosiello/catalog-go.git](https://github.com/lrosiello/catalog-go.git)
cd catalog-go
Configurar las variables de entorno:
Crea un archivo .env basado en .env.example:

Fragmento de código
PORT=8080
DATABASE_URL=postgres://usuario:password@localhost:5432/tubasedatos?sslmode=disable
JWT_SECRET=tu_secreto_jwt_super_seguro
WHATSAPP_NUMBER=5492490000000
Ejecutar migraciones:
Aplica el script ubicado en migrations/0001_init.sql sobre tu base de datos PostgreSQL.

Ejecutar la aplicación:

Bash
go run cmd/api/main.go
👨‍💻 Autor
Luis Antonio Rosiello

Backend Developer (Go & Java)
