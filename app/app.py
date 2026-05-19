import streamlit as st
import requests
from utils.api_client import fetch_system_health

# Page config
st.set_page_config(
    page_title="Chantry Suite",
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

.step-card {
    background: rgba(30, 41, 59, 0.2);
    border-left: 4px solid #6366F1;
    border-radius: 8px;
    padding: 15px;
    margin-bottom: 12px;
}
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

# Sidebar
st.sidebar.image("https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=300&q=80", use_container_width=True)
st.sidebar.markdown("<h2 style='text-align: center;'>Chantry Suite</h2>", unsafe_allow_html=True)
st.sidebar.markdown("---")
st.sidebar.info("🚀 **Dica:** Use as abas laterais para navegar entre a central de configuração, provisionamento, relatórios e comunicação.")

# Main Header
st.markdown("<h1 class='main-header'>Chantry Suite</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>O seu ecossistema automatizado de LMS e Gestão no Discord.</p>", unsafe_allow_html=True)

# Fetch System Health (Health Check API)
system_data = fetch_system_health()

if not system_data:
    st.error("🚨 Backend Offline. Verifique se o Go Daemon está a correr.")
    st.stop()

# If online, show success notification
st.success("🟢 Sistema Operacional e Sincronizado")

# Services Status Cards
services = system_data.get("services", {})
go_status = services.get("go_daemon", "offline")
pb_status = services.get("pocketbase", "offline")
discord_status = services.get("discord_ws", "disconnected")

st.markdown("### 🖥️ Status dos Serviços")
scol1, scol2, scol3 = st.columns(3)

with scol1:
    go_badge = "<span class='badge badge-online'>Healthy</span>" if go_status == "healthy" else "<span class='badge badge-offline'>Unhealthy</span>"
    st.markdown(f"""
    <div class='card'>
        <div class='card-title'>🐹 Go Daemon {go_badge}</div>
        <div class='card-desc'>
            Motor principal em Go que gerencia o fluxo de sincronização, provisionamento de canais privados e triggers assíncronos.
        </div>
    </div>
    """, unsafe_allow_html=True)

with scol2:
    pb_badge = "<span class='badge badge-online'>Healthy</span>" if pb_status == "healthy" else "<span class='badge badge-offline'>Unhealthy</span>"
    st.markdown(f"""
    <div class='card'>
        <div class='card-title'>🗄️ PocketBase {pb_badge}</div>
        <div class='card-desc'>
            Banco de dados SQLite embarcado. Armazena guildas configuradas, cargos monitorados, dados de alunos e histórico de presenças.
        </div>
    </div>
    """, unsafe_allow_html=True)

with scol3:
    discord_badge = "<span class='badge badge-online'>Connected</span>" if discord_status == "connected" else "<span class='badge badge-offline'>Disconnected</span>"
    st.markdown(f"""
    <div class='card'>
        <div class='card-title'>🤖 Discord WebSocket {discord_badge}</div>
        <div class='card-desc'>
            Conexão em tempo real (Gateway) do Bot para escuta de interações com botões de presença e sincronização ativa de canais.
        </div>
    </div>
    """, unsafe_allow_html=True)

# Global Metrics
metrics = system_data.get("metrics", {})
total_guilds = metrics.get("total_guilds", 0)
total_students = metrics.get("total_students", 0)
total_attendances = metrics.get("total_attendances", 0)

st.markdown("### 📊 Métricas Globais")
mcol1, mcol2, mcol3 = st.columns(3)
with mcol1:
    st.metric("Total de Servidores", total_guilds)
with mcol2:
    st.metric("Total de Alunos", total_students)
with mcol3:
    st.metric("Presenças Registradas", total_attendances)

# Installation Wizard
st.divider()
st.markdown("### 🤖 Adicionar Chantry a um Novo Servidor")

with st.container(border=True):
    client_id = system_data.get("env", {}).get("discord_client_id", "")
    oauth_url = f"https://discord.com/oauth2/authorize?client_id={client_id}&permissions=8&integration_type=0&scope=bot+applications.commands"
    
    st.write("Siga o assistente rápido para habilitar as funcionalidades de gerenciamento do Chantry no seu servidor do Discord:")
    
    st.markdown("""
    <div class='step-card'>
        <b>1. Convite de Instalação:</b> Clique no botão de convite abaixo para adicionar o Bot ao seu servidor do Discord com permissões administrativas completas.
    </div>
    <div class='step-card'>
        <b>2. Configuração de Cargos & Turnos:</b> Uma vez adicionado, navegue até a aba lateral de <b>Configurações</b> para selecionar os cargos das turmas que serão monitoradas.
    </div>
    <div class='step-card'>
        <b>3. Sincronização Inicial:</b> Vá em <b>Sincronização</b> para puxar automaticamente todos os membros ativos do cargo e registrá-los como alunos no banco.
    </div>
    """, unsafe_allow_html=True)
    
    if client_id:
        st.link_button("✨ Convidar Bot para o Discord", url=oauth_url, type="primary")
    else:
        st.warning("⚠️ Client ID do Discord não detectado no backend. Verifique a variável DISCORD_APP_ID no arquivo .env.")
