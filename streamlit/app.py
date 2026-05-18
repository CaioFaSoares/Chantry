import streamlit as st
import requests
import json
import socket

# Page config
st.set_page_config(
    page_title="Chantry Orquestração & POC Sandbox",
    page_icon="🌌",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Custom CSS with premium gradients, Outfit font, and glassmorphic cards
custom_css = """
<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap');

/* Apply modern font */
html, body, [class*="css"], .stMarkdown, .stButton button {
    font-family: 'Outfit', sans-serif !important;
}

/* Gradient Header */
.main-header {
    background: linear-gradient(135deg, #6366F1 0%, #A855F7 50%, #EC4899 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    font-weight: 700;
    font-size: 3rem;
    margin-bottom: 0.2rem;
    letter-spacing: -0.5px;
}

.subtitle {
    color: #94A3B8;
    font-size: 1.15rem;
    margin-bottom: 2rem;
    font-weight: 300;
}

/* Glassmorphic card design */
.card {
    background: rgba(30, 41, 59, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
    padding: 24px;
    backdrop-filter: blur(12px);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.2);
    margin-bottom: 20px;
    height: 100%;
}

.card:hover {
    transform: translateY(-4px);
    border-color: rgba(99, 102, 241, 0.4);
    box-shadow: 0 12px 30px rgba(99, 102, 241, 0.15);
}

.card-title {
    font-size: 1.3rem;
    font-weight: 600;
    color: #F8FAFC;
    margin-bottom: 8px;
    display: flex;
    align-items: center;
    gap: 8px;
}

.card-desc {
    color: #94A3B8;
    font-size: 0.95rem;
    line-height: 1.6;
    margin-bottom: 15px;
}

/* Custom premium status badges */
.badge {
    display: inline-flex;
    align-items: center;
    padding: 4px 12px;
    border-radius: 20px;
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.badge-online {
    background-color: rgba(16, 185, 129, 0.12);
    color: #10B981;
    border: 1px solid rgba(16, 185, 129, 0.25);
}

.badge-offline {
    background-color: rgba(244, 63, 94, 0.12);
    color: #F43F5E;
    border: 1px solid rgba(244, 63, 94, 0.25);
}

/* Custom modern button link */
.btn-action {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #6366F1 0%, #4F46E5 100%);
    color: #FFFFFF !important;
    text-decoration: none !important;
    padding: 10px 22px;
    border-radius: 10px;
    font-weight: 600;
    font-size: 0.9rem;
    transition: all 0.2s ease-in-out;
    border: none;
    box-shadow: 0 4px 14px rgba(79, 70, 229, 0.25);
}

.btn-action:hover {
    background: linear-gradient(135deg, #4F46E5 0%, #4338CA 100%);
    box-shadow: 0 6px 20px rgba(79, 70, 229, 0.4);
    transform: translateY(-1.5px);
}

.btn-secondary {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    color: #F8FAFC !important;
    text-decoration: none !important;
    padding: 10px 22px;
    border-radius: 10px;
    font-weight: 500;
    font-size: 0.9rem;
    transition: all 0.2s ease-in-out;
    border: 1px solid rgba(255, 255, 255, 0.1);
}

.btn-secondary:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.2);
    transform: translateY(-1.5px);
}
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

# Helper function to check if a port is open locally
def check_service_status(host, port):
    try:
        # We try opening a socket connection as a lightweight check
        socket.setdefaulttimeout(0.5)
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            if s.connect_ex((host, port)) == 0:
                return True
    except Exception:
        pass
    return False

# Sidebar
st.sidebar.image("https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=300&q=80", width=True)
st.sidebar.markdown("<h2 style='text-align: center;'>Chantry Suite</h2>", unsafe_allow_html=True)
st.sidebar.markdown("---")
st.sidebar.info("🚀 **Dica:** Ative o Docker Desktop e rode `docker compose up` para subir todos os serviços de uma vez.")

# Main Header
st.markdown("<h1 class='main-header'>Chantry Orchestration</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>Console unificado para prototipagem de POCs e fluxos automatizados de dados</p>", unsafe_allow_html=True)

# Check active statuses (checks internal Docker name first, then maps to local 12XXX safe ports)
pocketbase_online = check_service_status("pocketbase", 8090) or check_service_status("localhost", 12090)
go_online = check_service_status("go-server", 12000) or check_service_status("localhost", 12000)
streamlit_online = True # We are currently running it!

# Status Cards Layout (Unified grid of 3 columns)
col1, col2, col3 = st.columns(3)

with col1:
    badge_html = "<span class='badge badge-online'>Online</span>" if streamlit_online else "<span class='badge badge-offline'>Offline</span>"
    st.markdown(f"""
    <div class='card'>
        <div class='card-title'>💻 Streamlit Dashboard {badge_html}</div>
        <div class='card-desc'>
            Interface principal do Chantry construída para visualizações dinâmicas, validação rápida de hipóteses e central de comandos de APIs.
        </div>
        <p style='color:#64748B; font-size:0.85rem; margin-bottom: 20px;'>Porta Host: <b>12501</b> | Caminho: <code>./streamlit/app.py</code></p>
        <a href="#" class="btn-secondary" style="pointer-events: none; opacity: 0.6;">Você já está aqui</a>
    </div>
    """, unsafe_allow_html=True)

with col2:
    badge_html = "<span class='badge badge-online'>Online</span>" if pocketbase_online else "<span class='badge badge-offline'>Offline</span>"
    action_btn = '<a href="http://localhost:12090/_/" target="_blank" class="btn-action">Abrir Painel Admin</a>' if pocketbase_online else '<a href="http://localhost:12090/_/" target="_blank" class="btn-secondary">Tentar Acessar</a>'
    st.markdown(f"""
    <div class='card'>
        <div class='card-title'>🗄️ PocketBase Backend {badge_html}</div>
        <div class='card-desc'>
            Banco de dados relacional super leve e em tempo real. Inclui autenticação robusta, APIs automáticas e painel administrativo intuitivo embutido.
        </div>
        <p style='color:#64748B; font-size:0.85rem; margin-bottom: 20px;'>Porta Host: <b>12090</b> | Persistência: <code>pb_data</code> (SQLite & Files)</p>
        {action_btn}
    </div>
    """, unsafe_allow_html=True)

with col3:
    badge_html = "<span class='badge badge-online'>Online</span>" if go_online else "<span class='badge badge-offline'>Offline</span>"
    action_btn = '<a href="http://localhost:12000/api/health" target="_blank" class="btn-action">Abrir Healthcheck</a>' if go_online else '<a href="http://localhost:12000/api/health" target="_blank" class="btn-secondary">Tentar Acessar</a>'
    st.markdown(f"""
    <div class='card'>
        <div class='card-title'>🐹 Go Backend Daemon {badge_html}</div>
        <div class='card-desc'>
            Serviço core em Go de alta performance. Responsável pelas integrações com Discord API, execução de cronjobs agendados e mutações no PocketBase.
        </div>
        <p style='color:#64748B; font-size:0.85rem; margin-bottom: 20px;'>Porta Host: <b>12000</b> | Tecnologia: <code>Go + Fiber</code></p>
        {action_btn}
    </div>
    """, unsafe_allow_html=True)

st.markdown("---")
