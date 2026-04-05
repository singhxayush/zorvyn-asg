# Finance Data Processing & Access Control Backend

A logically structured Go backend built with the **Gin Framework** and **SQLite**. This system provides a financial dashboard API with Role-Based Access Control (RBAC), cookie-based JWT authentication, rate limiting for API spam protection and automated data migrations.

## 🚀 Getting Started

### 1. Clone and Setup

First, clone the repository and navigate into the project directory:

```bash
git clone https://github.com/singhxayush/zorvyn-asg.git
cd zorvyn-asg
```

Install the required Go dependencies:

```bash
go mod tidy
```

### 2. Configure Environment Variables

Create a `.env` file in the root directory. This project requires specific variables for the database, JWT security, and the initial admin setup:

```env
PORT=8080
APP_ENV=local
DB_URL=finance.db

# Security
JWT_SECRET=your_super_secret_key_here

# Initial Admin Seeding
ADMIN_USERNAME=superadmin
ADMIN_EMAIL=admin@test.com
ADMIN_PASSWORD=securepassword123
```

### 3. Explore Available Commands

You can view all available automation commands by running:

```bash
make help
```

### 4. Database Migrations

The project uses `golang-migrate` to manage the SQLite schema.

**Prerequisite:** The migration tool must be compiled with the `sqlite3` build tag. If you do not have it or have a version without SQLite support, run:

```bash
make install-migrate
```

Once installed, apply the database schema:

```bash
# Create the tables (users, financial_records)
make db-migrate-up

# To rollback if needed:
# make db-migrate-down
```

### 5. Initializing the Superadmin (Seeding)

Public registration is disabled for security. To create your first **Admin** user (using the credentials provided in your `.env`), run the seeding script:

```bash
make seed
```

### 6. Running the Application

You can now build or run the application directly.

**For development (with Live Reload):**
_Requires `air` installed._

```bash
make watch
```

**For standard execution:**

```bash
make run
```

**To build a production binary:**

```bash
make build
./bin/main
```

---

## 🔐 Access Control Logic

The system strictly enforces permissions based on three roles:

| Action                            | Viewer | Analyst | Admin |
| :-------------------------------- | :----: | :-----: | :---: |
| View Dashboard Summary            |   ✅   |   ✅    |  ✅   |
| List/Read Financial Records       |   ❌   |   ✅    |  ✅   |
| Create/Update/Delete Records      |   ❌   |   ❌    |  ✅   |
| Manage Users (Create/Status/Role) |   ❌   |   ❌    |  ✅   |

### Authentication Method

- **Cookie-based JWT**: The server issues an `access_token` (15 mins) and a `refresh_token` (7 days).
- **Security**: Tokens are stored in `HttpOnly` cookies to protect against XSS attacks.

---

## 🛠 Project Structure

- `cmd/api/main.go`: Application entry point.
- `cmd/seed/main.go`: One-time script to initialize the Admin.
- `internal/database/`: Database logic, raw SQL queries, and migrations.
- `internal/server/`: Gin handlers, routes, and RBAC middleware and rate limiter.
- `migrations/`: Versioned `.sql` files for schema evolution.

---

Here is the updated **Features Implemented** section for your `README.md`. I have added the **Rate Limiting** feature to the list:

---

## 📊 Features Implemented

- **Soft Deletes**: Financial records are never hard-deleted to preserve history.
- **Pagination**: Record fetching uses `limit/offset` (50 per page) to ensure high performance.
- **Data Integrity**: SQLite constraints (`CHECK`) ensure only valid roles, statuses, and transaction types are stored.
- **Secure Hashing**: User passwords are encrypted using `bcrypt` with a recommended cost factor.
- **Rate Limiting**: Protects the API from brute-force and DDoS attacks using a per-IP Token Bucket limiter (5 req/s with a burst of 10).
- **Cookie-based Auth**: Enhanced security via `HttpOnly` and `SameSite` cookies to prevent XSS-based token theft.
