# Cookie Syncer - Cloudflare Worker Backend

This directory contains the Cloudflare Worker implementation of the Cookie Syncer backend API. It is designed to be a feature-complete, serverless alternative to the original Go backend, leveraging the Cloudflare ecosystem for easy, free, and robust deployment for personal use.

## 1. Core Technologies

This implementation faithfully replicates the Go backend's functionality using a modern, serverless-native tech stack:

-   **Runtime**: [**Cloudflare Workers**](https://workers.cloudflare.com/) provides the serverless execution environment, running JavaScript/TypeScript at the network edge.
-   **API Framework**: [**Hono**](https://hono.dev/) is used as a fast and lightweight web framework for routing and handling HTTP requests, similar to `chi` in the Go version.
-   **Database**: [**Cloudflare D1**](https://developers.cloudflare.com/d1/), a serverless SQL database based on SQLite, serves as the persistent data store, directly replacing the local `cookiesyncer.db` file.
-   **Deployment & Management**: [**Wrangler CLI**](https://developers.cloudflare.com/workers/wrangler/) is used for all aspects of deployment, configuration, and database migration.
-   **Language**: **TypeScript** is used for its strong typing, which helps maintain code quality and prevent common errors.

## 2. Architecture & Implementation Highlights

The goal was to create a 1:1 functional equivalent of the Go backend. Here are the key architectural decisions and implementation details that achieved this:

### 2.1. Database Schema Compatibility

This was the most critical aspect of the migration. The final D1 database schema, defined in the `migrations/` directory, is **fully compatible** with the original SQLite schema used by the Go backend.

-   **Data Types**: All data types were precisely matched. For instance, `DATETIME` fields are stored as `TEXT` (ISO 8601 strings), and `BOOLEAN` fields are stored as `INTEGER` (`0` for `false`, `1` for `true`).
-   **Migrations**: Database schema changes are managed through SQL files in the `migrations/` directory and applied via the `wrangler d1 migrations apply` command.

### 2.2. API Endpoint & Logic Equivalence

All API endpoints from the Go backend have been re-implemented.

-   **Routing**: `Hono` is used to define all API routes, including user-facing routes (`/api/v1/*`) and admin-only routes (`/api/v1/admin/*`), mirroring the structure in `router.go`.
-   **Authentication**: The custom `x-api-key` header-based authentication is replicated in `src/index.ts`. The middleware fetches the user from the D1 database based on the provided key.
-   **Standardized Responses**: A `src/response.ts` helper module was created to ensure all JSON responses follow the Go backend's standard `{"code": ..., "message": ..., "data": ...}` format.
-   **Data Transformation**:
    -   **Presenter Layer (`src/presenter.ts`)**: This layer is responsible for converting data from the "database format" (e.g., `sharing_enabled: 0`) to the "API format" (e.g., `sharing_enabled: false`) before sending it to the client, ensuring perfect compatibility.
    -   **Client-Side (`background.ts`)**: The client-side `transformCookieForAPI` function was reverted to its original state, sending data in the exact format the Go backend expects. **No Base64 encoding is used**, as the root cause of sync failures was identified as data type mismatch, not WAF interference.

### 2.3. Initial Admin User Creation (Bootstrap)

To solve the "chicken-and-egg" problem of creating the first admin user, we replicated the Go backend's bootstrap mechanism:

-   A special, **unprotected** endpoint, `POST /api/v1/admin/init`, was created.
-   This endpoint checks if an admin user already exists in the D1 database.
-   If no admin exists, it creates one and returns its details, including the crucial initial API key.
-   If an admin already exists, it returns a `409 Conflict` error to prevent duplicates.

## 3. Deployment and Management Guide

### Step 1: Prerequisites

-   A [Cloudflare account](https://dash.cloudflare.com/sign-up).
-   [Node.js](https://nodejs.org/) and `npm` installed.
-   Log in to Wrangler: `npx wrangler login`.

### Step 2: Install Dependencies

Navigate to this directory (`api_service/cf`) and run:
```sh
npm install
```

### Step 3: Create and Prepare the D1 Database

This only needs to be done once.

1.  **Create the D1 Database**: This command provisions a new D1 database in your Cloudflare account.
    ```sh
    npm run d1:create
    ```
2.  **Update Configuration**: Wrangler will output a `database_id`. Copy this ID and paste it into the `database_id` field in your `wrangler.toml` file.
3.  **Apply Schema Migrations**: This command executes all SQL files in the `migrations/` directory against your production D1 database, creating the `users` and `cookies` tables.
    ```sh
    npm run d1:migrate:prod
    ```

### Step 4: Deploy the Worker

This command bundles and uploads your code to the Cloudflare network.

```sh
npm run deploy
```
After deployment, Wrangler will display the public URL of your worker (e.g., `https://cookie-syncer-api.<your-subdomain>.workers.dev`).

### Step 5: Create the Initial Admin User

Since the database is new and empty, you must create the first admin user.

1.  **Run the Init Command**: Use a tool like `curl` or `Invoke-WebRequest` to send a `POST` request to the `/init` endpoint.

    **For PowerShell:**
    ```powershell
    Invoke-WebRequest -Uri https://<YOUR_WORKER_URL>/api/v1/admin/init -Method POST
    ```

    **For bash/zsh (Linux, macOS, Git Bash):**
    ```sh
    curl -X POST https://<YOUR_WORKER_URL>/api/v1/admin/init
    ```

2.  **Save Your API Key**: The command will return a JSON object containing your new admin user's details. **Copy the `api_key` value immediately and store it securely.** This is your master key for the service.

### Step 6: Configure the Browser Extension

-   Open the Cookie Syncer extension's settings.
-   Set the "API Endpoint" to your Worker's public URL.
-   Set the "Auth Token" to the `api_key` you just saved.
-   Test the connection. It should now succeed.

Your serverless backend is now fully operational.
