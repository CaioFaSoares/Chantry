# Codebase App Report: Chantry Intelligence Hub

## Overview
The frontend is a [Streamlit](https://streamlit.io/) application written in Python. It provides a user-friendly interface for administrators to manage the Discord community, monitor attendance, and send broadcasts.

## Structure (`/app`)
- **`app.py`:** The main entry point and sidebar navigation.
- **`utils/api_client.py`:** The centralized communication layer. It handles:
    - **Backend Discovery:** Automatically switches between `http://go-server:12000` (Docker) and `http://localhost:12000` (Local dev).
    - **Caching:** Extensive use of `@st.cache_data` to minimize API calls and improve performance.
    - **Error Handling:** Graceful fallbacks when the backend is unreachable.
- **`pages/`:**
    - `2_server_setup.py`: Discord synchronization and private channel provisioning.
    - `3_intelligence_center.py`: Attendance dashboards and data export.
    - `4_engagement_hub.py`: Scheduling and managing broadcasts.

## UI/UX Patterns
- **Health Indicators:** Displays the status of the Go Daemon and Discord connection in the sidebar.
- **Interactive Forms:** Uses Streamlit components for role selection, time input, and broadcast drafting.
- **Real-time Feedback:** Cache purging ensures that UI updates immediately after a successful POST/PATCH operation.

## Key Utilities
- `get_backend_url()`: Resolves the Go server address based on the environment.
- `fetch_attendances()`: Retrieves attendance data with a low TTL (30s) for high reactivity.
- `sync_advanced_to_db()`: Orchestrates multi-step synchronization processes.
