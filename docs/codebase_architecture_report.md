# Codebase Architecture Report: System Design

## High-Level Architecture
Chantry is designed as a modular system where the frontend and backend are decoupled, communicating via a RESTful API.

### Components
1. **Frontend (Streamlit):** Python UI for administration.
2. **Backend (Fiber Go):** Orchestrator and "BFF" (Backend for Frontend).
3. **Database (PocketBase):** Go-based BaaS (Backend-as-a-Service) providing SQLite storage, Auth, and a real-time Admin UI.
4. **Integration (Discord):** External service managed via the Go server.

## Data Flow
1. **Admin Action:** User interacts with Streamlit.
2. **API Request:** Streamlit calls the Go Daemon's REST API.
3. **Logic Execution:** Go Daemon validates the request, updates PocketBase, and/or calls the Discord API.
4. **Feedback:** Go Daemon returns the result to Streamlit, which updates the UI.
5. **Background Jobs:** Go Daemon runs background tickers that monitor PocketBase state and trigger Discord interactions (e.g., attendance buttons).

## Deployment (Docker)
The system is orchestrated using `docker-compose.yml`, typically involving:
- `go-server`: Running the Go Daemon (port 12000).
- `streamlit-app`: Running the Python UI (port 8501).
- `pocketbase`: (Often embedded or run as a separate container).

## Security & Reliability
- **Environment Variables:** Credentials (Discord Token, PB Admin) are managed via `.env`.
- **Timezone Management:** Centralized timezone handling (default `America/Sao_Paulo`) ensures consistent scheduling between the UI and background workers.
- **Graceful Failures:** The system is designed to handle Discord API rate limits and intermittent database connectivity.
