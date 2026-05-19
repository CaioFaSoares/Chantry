import streamlit as st
import requests
import socket
import pandas as pd
from datetime import datetime

# 1. Page Config
st.set_page_config(
    page_title="Dashboard Diário de Presenças - Chantry",
    page_icon="📊",
    layout="wide",
    initial_sidebar_state="expanded"
)

# 2. Premium CSS (Outfit font, vibrant gradients, and elegant glassmorphism)
custom_css = """
<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap');

html, body, [class*="css"], .stMarkdown, .stButton button {
    font-family: 'Outfit', sans-serif !important;
}

.main-header {
    background: linear-gradient(135deg, #3B82F6 0%, #8B5CF6 50%, #EC4899 100%);
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

.card-section {
    background: rgba(30, 41, 59, 0.45);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
    padding: 24px;
    backdrop-filter: blur(12px);
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.25);
    margin-bottom: 24px;
}

.section-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: #F8FAFC;
    margin-bottom: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
}

.clock-badge {
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.15) 0%, rgba(139, 92, 246, 0.15) 100%);
    border: 1px solid rgba(59, 130, 246, 0.3);
    color: #60A5FA;
    padding: 10px 16px;
    border-radius: 12px;
    font-weight: 500;
    font-size: 1.05rem;
    margin-bottom: 24px;
    display: inline-flex;
    align-items: center;
    gap: 8px;
}
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

from utils.api_client import (
    fetch_server_health,
    get_server_timezone,
    fetch_guilds,
    fetch_guild_roles_config,
    fetch_attendances
)

# Initialize global session state
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None

# 4. Main UI Layout
st.markdown('<h1 class="main-header">📊 Dashboard Diário de Presenças</h1>', unsafe_allow_html=True)
st.markdown('<p class="subtitle">Acompanhe em tempo real a entrada, saída e o retrato operacional de presença da sua comunidade</p>', unsafe_allow_html=True)

# 5. Server Health Monitor
health = fetch_server_health()
if health and "timestamp" in health:
    try:
        dt_str = health["timestamp"]
        dt = datetime.fromisoformat(dt_str)
        formatted_time = dt.strftime("%H:%M:%S (%d/%m/%Y)")
        tz_name = get_server_timezone()
        
        st.markdown(
            f'<div class="clock-badge">⏰ <b>Horário do Servidor:</b> {formatted_time} &nbsp;|&nbsp; 🌍 <b>Timezone:</b> {tz_name}</div>',
            unsafe_allow_html=True
        )
    except Exception:
        tz_name = get_server_timezone()
        st.markdown(
            f'<div class="clock-badge">⏰ <b>Horário do Servidor:</b> {health["timestamp"]} &nbsp;|&nbsp; 🌍 <b>Timezone:</b> {tz_name}</div>',
            unsafe_allow_html=True
        )
else:
    st.markdown(
        '<div class="clock-badge" style="color: #EF4444; border-color: rgba(239, 68, 68, 0.3); background: rgba(239, 68, 68, 0.1);">⚠️ <b>Status do Daemon Go:</b> Offline ou inacessível</div>',
        unsafe_allow_html=True
    )

# 6. Sidebar Refresh controls
st.sidebar.markdown("### 🛠️ Controles de Atualização")
if st.sidebar.button("🔄 Atualizar Relatório"):
    st.rerun()

# 7. Core Workflow
guilds = fetch_guilds()

if guilds is None:
    st.error("❌ Não foi possível se conectar ao Chantry Go Daemon. Por favor, verifique se a API está online.")
elif not guilds:
    st.warning("⚠️ Nenhum servidor de Discord (Guild) mapeado. Certifique-se de conectar o bot a um servidor ativo.")
else:
    # Filter Cards
    st.markdown('<div class="card-section">', unsafe_allow_html=True)
    st.markdown('<div class="section-title">🔍 Parâmetros de Filtro do Painel</div>', unsafe_allow_html=True)
    
    col1, col2, col3 = st.columns(3)
    
    with col1:
        guild_options = {g["id"]: g["name"] for g in guilds}
        
        default_index = 0
        if st.session_state.selected_guild_id and st.session_state.selected_guild_id in guild_options:
            default_index = list(guild_options.keys()).index(st.session_state.selected_guild_id)

        selected_guild_id = st.selectbox(
            "Selecione o Servidor (Guild)",
            options=list(guild_options.keys()),
            index=default_index,
            format_func=lambda x: guild_options[x]
        )
        st.session_state.selected_guild_id = selected_guild_id
        
    with col2:
        roles_config = fetch_guild_roles_config(selected_guild_id)
        monitored_roles = [r for r in roles_config if r.get("is_monitored", False)]
        
        if not monitored_roles:
            st.warning("⚠️ Nenhuma turma monitorada neste servidor.")
            selected_role_id = None
        else:
            role_options = {r["id"]: r["name"] for r in monitored_roles}
            selected_role_id = st.selectbox(
                "Selecione a Turma (Role)",
                options=list(role_options.keys()),
                format_func=lambda x: role_options[x]
            )
            
    with col3:
        selected_date = st.date_input("Selecione a Data", value=datetime.today())
        date_str = selected_date.strftime("%Y-%m-%d")
        
    st.markdown('</div>', unsafe_allow_html=True)
    
    if selected_role_id:
        # Fetch data
        attendances = fetch_attendances(selected_guild_id, date_str, selected_role_id)
        
        # 8. Analytical Metrics Header
        completed_count = sum(1 for a in attendances if a.get("status") == "completed")
        pending_count = sum(1 for a in attendances if a.get("status") == "pending_checkout")
        late_count = sum(1 for a in attendances if a.get("status") == "late")
        absent_count = sum(1 for a in attendances if a.get("status") == "absent")
        total_count = len(attendances)
        
        m_col1, m_col2, m_col3, m_col4, m_col5 = st.columns(5)
        
        with m_col1:
            st.metric("👥 Total Registrado", value=total_count)
        with m_col2:
            st.metric("🟢 Completos (Entrada + Saída)", value=completed_count)
        with m_col3:
            st.metric("🟡 Em Andamento (Falta Saída)", value=pending_count)
        with m_col4:
            st.metric("🟠 Atrasados", value=late_count)
        with m_col5:
            st.metric("🔴 Faltas", value=absent_count)
            
        # 9. Attendance Table
        st.markdown('<div class="card-section">', unsafe_allow_html=True)
        st.markdown(f'<div class="section-title">📋 Quadro de Presenças ({selected_date.strftime("%d/%m/%Y")})</div>', unsafe_allow_html=True)
        
        if not attendances:
            st.info("ℹ️ Nenhuma presença registrada para os parâmetros selecionados nesta data.")
        else:
            # Build DataFrame
            data_list = []
            for a in attendances:
                # Localize date/time format elegantly
                clock_in_raw = a.get("clock_in", "")
                clock_out_raw = a.get("clock_out", "")
                
                def parse_and_format_time(raw_time):
                    if not raw_time:
                        return "--:--"
                    try:
                        # Handle multiple datetime formats safely
                        for fmt in ("%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%d %H:%M:%S.%fZ", "%Y-%m-%d %H:%M:%SZ", "%Y-%m-%d %H:%M:%S"):
                            try:
                                dt = datetime.strptime(raw_time[:19], fmt[:19])
                                return dt.strftime("%H:%M:%S")
                            except ValueError:
                                continue
                        return raw_time
                    except Exception:
                        return raw_time
                
                formatted_in = parse_and_format_time(clock_in_raw)
                formatted_out = parse_and_format_time(clock_out_raw)
                
                # Visual Status Mapping
                status_raw = a.get("status", "absent")
                status_mapped = "🔴 Falta"
                if status_raw == "completed":
                    status_mapped = "🟢 Completo"
                elif status_raw == "pending_checkout":
                    status_mapped = "🟡 Em andamento"
                elif status_raw == "late":
                    status_mapped = "🟠 Atrasado"
                elif status_raw == "justified":
                    status_mapped = "🔵 Justificado"
                
                source_raw = a.get("source", "discord_bot")
                source_mapped = "🤖 Discord Bot" if source_raw == "discord_bot" else "🛠️ Ajuste Manual"
                
                data_list.append({
                    "Aluno (Usuário)": a.get("student_name", ""),
                    "Nome de Exibição": a.get("student_nickname", ""),
                    "Status": status_mapped,
                    "Check-In": formatted_in,
                    "Check-Out": formatted_out,
                    "Canal de Entrada": source_mapped
                })
                
            df = pd.DataFrame(data_list)
            
            # Display DataFrame Premium columns configuration
            st.dataframe(
                df,
                use_container_width=True,
                column_config={
                    "Aluno (Usuário)": st.column_config.TextColumn(help="Nickname cadastrado do Discord"),
                    "Nome de Exibição": st.column_config.TextColumn(help="Nome real ou apelido da planilha"),
                    "Status": st.column_config.TextColumn(help="Estado atual da presença"),
                    "Check-In": st.column_config.TextColumn(help="Hora da entrada do ponto"),
                    "Check-Out": st.column_config.TextColumn(help="Hora da saída do ponto"),
                    "Canal de Entrada": st.column_config.TextColumn(help="Origem do registro")
                }
            )
            
        st.markdown('</div>', unsafe_allow_html=True)
