import streamlit as st
import pandas as pd
import time
from utils.api_client import (
    fetch_guilds,
    fetch_guild_roles_config,
    fetch_squad_dashboard_data,
    update_squad_channel
)

st.set_page_config(page_title="Gestão de Turmas - Chantry", page_icon="👥", layout="wide")

st.markdown("""
<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap');

html, body, [class*="css"], .stMarkdown, .stButton button {
    font-family: 'Outfit', sans-serif !important;
}

.main-header {
    background: linear-gradient(135deg, #4F46E5 0%, #06B6D4 100%);
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

.metric-card {
    background: rgba(30, 41, 59, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
    padding: 24px;
    backdrop-filter: blur(12px);
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.2);
    text-align: center;
    margin-bottom: 16px;
}
.metric-value {
    font-size: 2.5rem;
    font-weight: 700;
    color: #F8FAFC;
}
.metric-label {
    font-size: 0.95rem;
    color: #94A3B8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 500;
    margin-top: 8px;
}
</style>
""", unsafe_allow_html=True)

st.markdown("<h1 class='main-header'>👥 Hub de Gestão de Turmas</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>Vincule canais de comunicação, acompanhe o engajamento e gerencie os alunos de cada Squad.</p>", unsafe_allow_html=True)

# 1. Filters Setup
guilds = fetch_guilds()
if not guilds:
    st.warning("Nenhum servidor Discord configurado.")
    st.stop()

with st.container():
    col1, col2 = st.columns(2)
    
    with col1:
        guild_opts = {g['id']: g['name'] for g in guilds}
        selected_guild_id = st.selectbox(
            "Servidor (Guilda)", 
            options=list(guild_opts.keys()), 
            format_func=lambda x: guild_opts[x]
        )
        
    with col2:
        roles_config = fetch_guild_roles_config(guild_id=selected_guild_id)
        # Filter only monitored roles for this view
        monitored_roles = [r for r in roles_config if r.get("is_monitored", False)]
        
        if not monitored_roles:
            st.info("Nenhuma turma monitorada neste servidor. Configure as turmas primeiro na tela 'Configuração de Turnos'.")
            st.stop()
            
        role_opts = {r['id']: f"{r['name']} (Turno: {r.get('shift', 'N/A')})" for r in monitored_roles}
        selected_role_id = st.selectbox(
            "Turma (Squad)", 
            options=list(role_opts.keys()), 
            format_func=lambda x: role_opts[x]
        )

st.markdown("---")

# 2. Fetch Aggregated Data
if selected_role_id:
    with st.spinner("Carregando hub da turma..."):
        dashboard_data = fetch_squad_dashboard_data(selected_role_id)
        
    if not dashboard_data:
        st.error("Erro ao carregar dados da turma. Verifique a conexão com o servidor.")
    else:
        squad_info = dashboard_data.get("squad_info", {})
        metrics = dashboard_data.get("metrics", {})
        text_channels = dashboard_data.get("text_channels", [])
        students = dashboard_data.get("students", [])

        # --- Section 1: Channel Configuration ---
        st.markdown("### ⚙️ Canal Oficial da Turma")
        with st.container(border=True):
            col_ch, col_btn = st.columns([0.7, 0.3])
            
            # Map text channels for selectbox
            channel_opts = {"": "Nenhum canal vinculado"}
            for ch in text_channels:
                channel_opts[ch['id']] = f"#{ch['name']}"
                
            current_channel = squad_info.get("squad_channel_id", "")
            
            with col_ch:
                new_channel_id = st.selectbox(
                    "Selecione o canal de texto base desta turma no Discord:",
                    options=list(channel_opts.keys()),
                    format_func=lambda x: channel_opts[x],
                    index=list(channel_opts.keys()).index(current_channel) if current_channel in channel_opts else 0
                )
                
            with col_btn:
                st.write("") # spacing alignment
                st.write("")
                if st.button("💾 Vincular Canal", use_container_width=True, type="primary"):
                    if new_channel_id != current_channel:
                        with st.spinner("Salvando..."):
                            success, msg = update_squad_channel(selected_role_id, new_channel_id)
                            if success:
                                st.success("Canal vinculado com sucesso!")
                                time.sleep(1.0)
                                st.rerun()
                            else:
                                st.error(f"Erro ao salvar: {msg}")
                    else:
                        st.info("O canal selecionado já é o atual.")
                        
        st.markdown("<br>", unsafe_allow_html=True)
        
        # --- Section 2: Metrics Dashboard ---
        st.markdown("### 📊 Visão Geral")
        m1, m2, m3 = st.columns(3)
        
        m1.markdown(f"""
        <div class="metric-card">
            <div class="metric-value">{metrics.get("total_students", 0)}</div>
            <div class="metric-label">Total de Alunos</div>
        </div>
        """, unsafe_allow_html=True)
        
        m2.markdown(f"""
        <div class="metric-card">
            <div class="metric-value" style="color: #38BDF8;">{metrics.get("provisioned_1on1_channels", 0)}</div>
            <div class="metric-label">Canais 1-on-1 Criados</div>
        </div>
        """, unsafe_allow_html=True)
        
        rate = metrics.get("average_attendance_percent", 0.0)
        rate_color = "#34D399" if rate >= 75.0 else ("#FBBF24" if rate >= 50 else "#F87171")
        m3.markdown(f"""
        <div class="metric-card">
            <div class="metric-value" style="color: {rate_color};">{rate:.1f}%</div>
            <div class="metric-label">Média de Presença (30d)</div>
        </div>
        """, unsafe_allow_html=True)

        st.markdown("<br>", unsafe_allow_html=True)

        # --- Section 3: Students Roster ---
        st.markdown("### 🧑‍🎓 Lista de Alunos")
        if not students:
            st.info("Nenhum aluno ativo encontrado nesta turma.")
        else:
            df = pd.DataFrame(students)
            
            # Map visual statuses
            df["has_1on1_ui"] = df["has_1on1"].apply(lambda x: "🟢 Pronto" if x else "🔴 Pendente")
            
            df_display = df[["username", "nickname", "has_1on1_ui"]].copy()
            df_display.rename(columns={
                "username": "Discord Username",
                "nickname": "Nome / Apelido",
                "has_1on1_ui": "Status Canal 1-on-1"
            }, inplace=True)
            
            st.dataframe(df_display, use_container_width=True, hide_index=True)
