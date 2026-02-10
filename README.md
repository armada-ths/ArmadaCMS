# ArmadaCMS

Backend API and admin dashboard for [THS Armada](https://armada.nu). Provides REST endpoints consumed by the public website ([armada.nu](https://github.com/armada-ths/armada.nu)) and a React-Admin interface for content management.

## Tech Stack

### Backend

- **Language**: Go 1.24
- **Router**: [Gorilla Mux](https://github.com/gorilla/mux)
- **ORM**: [GORM](https://gorm.io/) (Postgres)
- **Auth**: JWT (Bearer tokens)
- **File storage**: AWS S3
- **Hot reload**: [Air](https://github.com/air-verse/air) (in Docker dev mode)

### Admin Frontend

- **Framework**: [React-Admin v5](https://marmelab.com/react-admin/)
- **Build tool**: [Vite](https://vitejs.dev/)
- **Language**: TypeScript
- **UI**: MUI (Material UI)

### Infrastructure

- **Deployment**: Docker on AWS
- **Database**: PostgreSQL (AWS RDS)

## Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) and npm (for the admin frontend)
- [Docker](https://www.docker.com/) and Docker Compose (optional, for containerized setup)
- PostgreSQL instance (local or remote)

## Getting Started

1. **Clone the repo**

   ```bash
   git clone https://github.com/armada-ths/ArmadaCMS.git
   cd ArmadaCMS
   ```

2. **Set up environment variables**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` with your Postgres credentials. See `.env.example` for all available variables and descriptions.

### Option A: Docker (recommended)

Builds the frontend and backend in one step:

```bash
docker compose up --build
```

> DB credentials can also be set directly in `docker-compose.yml` under `environment`.

### Option B: Run locally

1. **Build the admin frontend**

   ```bash
   cd frontend
   npm install
   npm run build
   cd ..
   ```

2. **Start the Go server**

   ```bash
   go run main.go
   ```

Once running, the server is available at:

- **App**: [http://localhost:8080](http://localhost:8080)
- **API**: [http://localhost:8080/api/v1/](http://localhost:8080/api/v1/)
- **Admin UI**: [http://localhost:8080/admin/](http://localhost:8080/admin/)
- **Health check**: [http://localhost:8080/health](http://localhost:8080/health)

### Frontend development

To develop the admin frontend with hot reload:

```bash
cd frontend
npm install
npm run dev
```

This starts a Vite dev server that proxies API requests to `localhost:8080`.

## Project Structure

```
ArmadaCMS/
├── main.go               # Entry point — routing, auto-migration, server startup
├── auth/
│   └── middleware.go      # JWT Bearer token auth middleware
├── Controllers/           # HTTP handlers (one per resource)
├── models/                # GORM model structs
├── db/
│   └── connect.go         # Postgres connection setup
├── utils/                 # Helpers (S3 upload, JWT, password hashing)
├── frontend/              # React-Admin SPA (Vite)
│   └── src/
│       ├── App.tsx            # Resource registrations
│       ├── dataProvider.ts    # Custom ra-data-simple-rest provider
│       ├── components/        # List, Create, Edit per resource
│       └── context/           # Auth provider, API endpoint config
├── Dockerfile             # Multi-stage build (frontend + Go + Air)
├── Dockerfile.prod        # Production multi-stage build
└── docker-compose.yml     # Docker Compose config
```

## API

All endpoints are under `/api/v1`. Routes are split into:

- **Public** (no auth): `GET` endpoints for resources like exhibitors, events, profiles, teams, dates.
- **Protected** (Bearer JWT): `POST`, `PUT`, `DELETE` and admin-only `GET` endpoints.

### Example endpoints

| Method   | Endpoint                  | Auth     | Description         |
| -------- | ------------------------- | -------- | ------------------- |
| `POST`   | `/api/v1/login`           | No       | Get JWT tokens      |
| `GET`    | `/api/v1/exhibitors`      | No       | List all exhibitors |
| `POST`   | `/api/v1/exhibitors`      | Required | Create exhibitor    |
| `PUT`    | `/api/v1/exhibitors/{id}` | Required | Update exhibitor    |
| `DELETE` | `/api/v1/exhibitors/{id}` | Required | Delete exhibitor    |
| `GET`    | `/api/v1/dates`           | No       | Get fair dates      |
| `GET`    | `/health`                 | No       | Health check        |

## Adding a New Resource

1. Create a model in `models/` with GORM struct tags and camelCase JSON tags.
2. Register the model in `db.DB.AutoMigrate(...)` in `main.go`.
3. Create a controller in `Controllers/` following existing CRUD patterns.
4. Add routes in `main.go` (public for reads, protected for writes).
5. Create `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
6. Register the `<Resource>` in `frontend/src/App.tsx`.
7. If the resource has file uploads, add it to the multipart list in `frontend/src/dataProvider.ts`.
