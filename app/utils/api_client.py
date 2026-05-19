import streamlit as st
import requests
import socket

# 1. Base URL Resolution with Container Fallback
def get_backend_url():
    try:
        socket.gethostbyname("go-server")
        return "http://go-server:12000/api"
    except socket.gaierror:
        return "http://localhost:12000/api"

base_url = get_backend_url()

# 2. Server Health Check (Cached, TTL = 10s to keep clock badge dynamic but lightweight)
@st.cache_data(ttl=10)
def fetch_server_health():
    try:
        resp = requests.get(f"{base_url}/health", timeout=3.0)
        return resp.json() if resp.status_code == 200 else None
    except Exception:
        return None

# 3. Read Operations (Cached, TTL = 300s / 5 minutes)
@st.cache_data(ttl=300)
def fetch_guilds(unused_url=None):
    # Keep unused_url argument to prevent breaking existing code during sync page refactoring
    try:
        resp = requests.get(f"{base_url}/discord/guilds", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

@st.cache_data(ttl=300)
def fetch_roles(unused_url=None, guild_id=None):
    # Support both (unused_url, guild_id) and (guild_id) signatures
    gid = guild_id if guild_id is not None else unused_url
    try:
        resp = requests.get(f"{base_url}/discord/guilds/{gid}/roles", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

@st.cache_data(ttl=300)
def fetch_guild_roles_config(guild_id):
    try:
        resp = requests.get(f"{base_url}/config/guilds/{guild_id}/roles", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

@st.cache_data(ttl=300)
def fetch_categories(guild_id):
    try:
        resp = requests.get(f"{base_url}/discord/guilds/{guild_id}/categories", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

@st.cache_data(ttl=300)
def fetch_members(unused_url=None, guild_id=None, role_id=None):
    # Support (unused_url, guild_id, role_id) signature
    gid = guild_id if guild_id is not None else unused_url
    try:
        resp = requests.get(f"{base_url}/discord/guilds/{gid}/members", timeout=10.0)
        if resp.status_code == 200:
            members = resp.json()
            if role_id:
                # Local filtering by role to maintain compatibility
                filtered = []
                for m in members:
                    roles_list = m.get("roles", [])
                    if role_id in roles_list:
                        filtered.append(m)
                return filtered
            return members
        return []
    except Exception:
        return []

@st.cache_data(ttl=300)
def fetch_guild_managers(guild_id):
    try:
        resp = requests.get(f"{base_url}/test/guilds/{guild_id}/managers", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

@st.cache_data(ttl=30, show_spinner=False) # Keep attendance dashboard extremely reactive
def fetch_attendances(guild_id, date_str, role_id):
    try:
        resp = requests.get(
            f"{base_url}/reports/guilds/{guild_id}/attendances",
            params={"date": date_str, "role_id": role_id},
            timeout=5.0
        )
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

# 4. Write Operations (POST/PATCH - Clears respective caches on success)
def create_category(guild_id, name, position):
    payload = {"name": name, "position": position}
    try:
        resp = requests.post(
            f"{base_url}/discord/guilds/{guild_id}/categories",
            json=payload,
            timeout=5.0
        )
        if resp.status_code == 201:
            fetch_categories.clear() # Purge categories cache
            return True, resp.json()
        return False, f"Status: {resp.status_code}"
    except Exception as e:
        return False, str(e)

def update_role_config(role_id, shift=None, check_in_time=None, checkout_cooldown=None, is_monitored=None, is_active=None):
    payload = {}
    if shift is not None:
        payload["shift"] = shift
    if check_in_time is not None:
        payload["check_in_time"] = check_in_time
    if checkout_cooldown is not None:
        payload["checkout_cooldown"] = checkout_cooldown
    if is_monitored is not None:
        payload["is_monitored"] = is_monitored
    if is_active is not None:
        payload["is_active"] = is_active

    try:
        resp = requests.patch(
            f"{base_url}/config/roles/{role_id}",
            json=payload,
            timeout=5.0
        )
        if resp.status_code == 200:
            fetch_guild_roles_config.clear() # Purge roles config cache!
            return True, resp.json()
        return False, f"Status: {resp.status_code}"
    except Exception as e:
        return False, str(e)

def provision_channels(guild_id, category_id, role_id):
    payload = {"category_id": category_id, "role_id": role_id}
    try:
        resp = requests.post(
            f"{base_url}/provision/guilds/{guild_id}/channels",
            json=payload,
            timeout=30.0 # Long timeout for channel creation
        )
        return resp.status_code == 200, resp.json()
    except Exception as e:
        return False, str(e)

def heal_channels(guild_id, category_id):
    payload = {"category_id": category_id}
    try:
        resp = requests.post(
            f"{base_url}/provision/guilds/{guild_id}/heal",
            json=payload,
            timeout=30.0
        )
        return resp.status_code == 200, resp.json()
    except Exception as e:
        return False, str(e)

@st.cache_data(ttl=300)
def get_provision_page_data(guild_id):
    try:
        resp = requests.get(f"{base_url}/ui/provision-page/{guild_id}", timeout=10.0)
        return resp.json() if resp.status_code == 200 else None
    except Exception:
        return None

def save_announcement_channel(guild_id, channel_id):
    payload = {"announcement_channel_id": channel_id}
    try:
        resp = requests.patch(
            f"{base_url}/config/guilds/{guild_id}",
            json=payload,
            timeout=5.0
        )
        if resp.status_code == 200:
            get_provision_page_data.clear() # Purge BFF cache on successful save
            return True, resp.json()
        return False, f"Status: {resp.status_code}"
    except Exception as e:
        return False, str(e)


def sync_advanced_to_db(unused_url=None, guild_id=None, payload=None):
    # Support both (unused_url, guild_id, payload) and (guild_id, payload) signatures
    gid = guild_id if guild_id is not None else unused_url
    pld = payload if payload is not None else guild_id
    try:
        resp = requests.post(
            f"{base_url}/sync/guilds/{gid}/advanced",
            json=pld,
            timeout=10.0
        )
        if resp.status_code == 200:
            # Synchronization alters members, managers and roles
            fetch_guilds.clear()
            fetch_roles.clear()
            fetch_guild_roles_config.clear()
            fetch_guild_managers.clear()
            return resp.json()
        return None
    except Exception:
        return None

def trigger_test_attendance(guild_id, channel_id, tester_discord_id):
    payload = {
        "guild_id": guild_id,
        "channel_id": channel_id,
        "tester_discord_id": tester_discord_id
    }
    try:
        resp = requests.post(
            f"{base_url}/test/attendance/trigger",
            json=payload,
            timeout=5.0
        )
        if resp.status_code == 200:
            # Testing triggers an attendance record mock insertion
            fetch_attendances.clear()
            return True, resp.json().get("message", "Sucesso")
        else:
            return False, resp.json().get("error", f"Status: {resp.status_code}")
    except Exception as e:
        return False, str(e)

# 5. Broadcast Operations
@st.cache_data(ttl=15, show_spinner=False)  # Lightweight cache for Broadcast UI BFF
def fetch_broadcast_page_data(guild_id):
    try:
        resp = requests.get(f"{base_url}/ui/broadcast-page/{guild_id}", timeout=10.0)
        return resp.json() if resp.status_code == 200 else None
    except Exception:
        return None

def schedule_broadcast(guild_id, content, target_type, target_roles, schedule_time):
    payload = {
        "guild_id": guild_id,
        "content": content,
        "target_type": target_type,
        "target_roles": target_roles,
        "schedule_time": schedule_time
    }
    try:
        resp = requests.post(f"{base_url}/broadcasts", json=payload, timeout=5.0)
        if resp.status_code == 201:
            fetch_broadcast_page_data.clear()  # Purge BFF page data cache on success!
            return True, resp.json()
        return False, resp.json().get("error", f"Status: {resp.status_code}")
    except Exception as e:
        return False, str(e)

def cancel_broadcast(broadcast_id):
    try:
        resp = requests.delete(f"{base_url}/broadcasts/{broadcast_id}", timeout=5.0)
        if resp.status_code == 200:
            fetch_broadcast_page_data.clear()  # Purge BFF page data cache on success!
            return True, resp.json().get("message", "Sucesso")
        return False, resp.json().get("error", f"Status: {resp.status_code}")
    except Exception as e:
        return False, str(e)

@st.cache_data(ttl=30, show_spinner=False)
def fetch_export_report(guild_id, start_date, end_date, role_id=None):
    params = {
        "start_date": start_date,
        "end_date": end_date
    }
    if role_id and role_id != "all":
        params["role_id"] = role_id

    try:
        resp = requests.get(
            f"{base_url}/reports/guilds/{guild_id}/export",
            params=params,
            timeout=10.0
        )
        return resp.json() if resp.status_code == 200 else None
    except Exception:
        return None

