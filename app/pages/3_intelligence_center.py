import streamlit as st
import pandas as pd
from datetime import datetime, date

# 1. Page Config
st.set_page_config(
    page_title="Centro de Inteligência e Relatórios - Chantry",
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

.metric-card {
    background-color: rgba(30, 41, 59, 0.4);
    border-radius: 16px;
    padding: 20px;
    text-align: center;
    border: 1px solid rgba(255, 255, 255, 0.08);
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.2);
}

.metric-value {
    font-size: 2rem;
    font-weight: bold;
    color: #FFFFFF;
}

.metric-label {
    font-size: 0.9rem;
    color: #94A3B8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 500;
    margin-top: 8px;
}
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

from utils.api_client import (
    fetch_server_health,
    get_server_timezone,
    fetch_guilds,
    fetch_guild_roles_config,
    fetch_attendances,
    fetch_export_report
)

# Initialize session state for guild selector
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None

# Header
st.markdown('<h1 class="main-header">📊 Centro de Inteligência e Relatórios</h1>', unsafe_allow_html=True)
st.markdown('<p class="subtitle">Analise presenças diárias, extraia relatórios consolidados e exporte planilhas BI</p>', unsafe_allow_html=True)

# Server Health & Clock
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

# Fetch Guilds list
guilds = fetch_guilds()

if guilds is None:
    st.error(
        "❌ **Erro de Conexão:** Não foi possível conectar ao Daemon do Go Server. "
        "Verifique se o serviço `go-server` está ativo na porta `12000`."
    )
    st.stop()

if not guilds:
    st.warning(
        "⚠️ **Nenhum Servidor Identificado:** O bot do Chantry não foi adicionado a nenhum "
        "servidor ou o token não possui permissões adequadas de leitura."
    )
    st.stop()

# 3. Global Guild Context Selector (Main Page Area)
st.markdown("<div class='card-section'>", unsafe_allow_html=True)
st.markdown("<div class='section-title'>🔌 Seleção do Servidor de Relatórios</div>", unsafe_allow_html=True)

guild_options = {g["id"]: g["name"] for g in guilds}
default_index = 0
if st.session_state.selected_guild_id and st.session_state.selected_guild_id in guild_options:
    default_index = list(guild_options.keys()).index(st.session_state.selected_guild_id)

selected_guild_id = st.selectbox(
    "Escolha o Servidor Discord para visualizar relatórios:",
    options=list(guild_options.keys()),
    index=default_index,
    format_func=lambda x: guild_options[x],
    key="global_reports_guild_selector"
)
st.session_state.selected_guild_id = selected_guild_id

st.markdown("</div>", unsafe_allow_html=True)

# Sidebar refreshes
st.sidebar.markdown("### 🛠️ Controles de Atualização")
if st.sidebar.button("🔄 Atualizar Dados", use_container_width=True):
    st.rerun()

# Set up Tabbed Interface
tab_daily, tab_historical = st.tabs([
    "📅 Visão Diária (Ponto)", 
    "📈 Visão Histórica & Exportação"
])

# ==========================================
# TAB 1: Visão Diária (Ponto)
# ==========================================
with tab_daily:
    st.markdown("### 📅 Presenças do Dia")
    st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top:-10px; margin-bottom: 20px;'>"
                "Monitore a entrada, saída e o status em tempo real de cada aluno.</p>",
                unsafe_allow_html=True)

    roles_config = fetch_guild_roles_config(selected_guild_id)
    monitored_roles = [r for r in roles_config if r.get("is_monitored", False)] if roles_config else []

    if not monitored_roles:
        st.warning("⚠️ **Nenhuma turma monitorada neste servidor.** Habilite o monitoramento de turmas na tela de Configuração do Servidor (Setup).")
    else:
        st.markdown('<div class="card-section">', unsafe_allow_html=True)
        st.markdown('<div class="section-title">🔍 Parâmetros de Filtro Diário</div>', unsafe_allow_html=True)

        col_d1, col_d2 = st.columns(2)
        with col_d1:
            role_options = {r["id"]: r["name"] for r in monitored_roles}
            selected_role_id = st.selectbox(
                "Selecione a Turma (Cargo)",
                options=list(role_options.keys()),
                format_func=lambda x: role_options[x],
                key="daily_role_select"
            )
        with col_d2:
            selected_date = st.date_input("Selecione a Data", value=date.today(), key="daily_date_select")
            date_str = selected_date.strftime("%Y-%m-%d")

        st.markdown('</div>', unsafe_allow_html=True)

        if selected_role_id:
            with st.spinner("Carregando presença diária..."):
                attendances = fetch_attendances(selected_guild_id, date_str, selected_role_id)

            if attendances is None:
                st.error("Erro ao buscar registros de presença no backend.")
            else:
                # Calculate metrics
                completed_count = sum(1 for a in attendances if a.get("status") == "completed")
                pending_count = sum(1 for a in attendances if a.get("status") == "pending_checkout")
                late_count = sum(1 for a in attendances if a.get("status") == "late")
                absent_count = sum(1 for a in attendances if a.get("status") == "absent")
                total_count = len(attendances)

                m_col1, m_col2, m_col3, m_col4, m_col5 = st.columns(5)
                with m_col1:
                    st.metric("👥 Total Registrado", value=total_count)
                with m_col2:
                    st.metric("🟢 Completos", value=completed_count)
                with m_col3:
                    st.metric("🟡 Em Andamento", value=pending_count)
                with m_col4:
                    st.metric("🟠 Atrasados", value=late_count)
                with m_col5:
                    st.metric("🔴 Faltas", value=absent_count)

                st.markdown("<br>", unsafe_allow_html=True)

                st.markdown('<div class="card-section">', unsafe_allow_html=True)
                st.markdown(f'<div class="section-title">📋 Quadro de Presenças ({selected_date.strftime("%d/%m/%Y")})</div>', unsafe_allow_html=True)

                if not attendances:
                    st.info("ℹ️ Nenhuma presença registrada para os parâmetros selecionados nesta data.")
                else:
                    data_list = []
                    for a in attendances:
                        clock_in_raw = a.get("clock_in", "")
                        clock_out_raw = a.get("clock_out", "")

                        def parse_and_format_time(raw_time):
                            if not raw_time:
                                return "--:--"
                            try:
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

                    st.dataframe(
                        df,
                        use_container_width=True,
                        column_config={
                            "Aluno (Usuário)": st.column_config.TextColumn(help="Username cadastrado do Discord"),
                            "Nome de Exibição": st.column_config.TextColumn(help="Nome real ou apelido do aluno"),
                            "Status": st.column_config.TextColumn(help="Estado atual da presença"),
                            "Check-In": st.column_config.TextColumn(help="Hora da entrada do ponto"),
                            "Check-Out": st.column_config.TextColumn(help="Hora da saída do ponto"),
                            "Canal de Entrada": st.column_config.TextColumn(help="Origem do registro")
                        }
                    )
                st.markdown('</div>', unsafe_allow_html=True)


# ==========================================
# TAB 2: Visão Histórica & Exportação Relatório BI
# ==========================================
with tab_historical:
    st.markdown("### 📈 Central de Relatórios e BI")
    st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top:-10px; margin-bottom: 20px;'>"
                "Extraia dados históricos e exporte planilhas de frequência para planilhamento externo.</p>",
                unsafe_allow_html=True)

    roles = fetch_guild_roles_config(guild_id=selected_guild_id)

    st.markdown('<div class="card-section">', unsafe_allow_html=True)
    st.markdown('<div class="section-title">🔍 Filtros de Consulta do Período</div>', unsafe_allow_html=True)

    col_h1, col_h2 = st.columns(2)
    with col_h1:
        role_opts = {"all": "🌍 Todas as Turmas / Roles"}
        if roles:
            for r in roles:
                role_opts[r['discord_id']] = f"{r['name']} (Turno: {r['shift']})"

        selected_role_id = st.selectbox(
            "Filtrar por Turma",
            options=list(role_opts.keys()),
            format_func=lambda x: role_opts[x],
            key="historical_role_select"
        )
    with col_h2:
        date_range = st.date_input("Selecione o Período (Data Início e Fim)", [], key="historical_date_range")

    st.write("")
    col_btn, _ = st.columns([1, 4])
    with col_btn:
        generate_btn = st.button("📊 Gerar Relatório", type="primary", use_container_width=True, key="historical_generate_btn")
    
    st.markdown('</div>', unsafe_allow_html=True)

    if generate_btn:
        if len(date_range) != 2:
            st.warning("⚠️ Por favor, selecione as datas de **início** e **fim** no calendário antes de gerar o relatório.")
        else:
            start_date_str = date_range[0].strftime("%Y-%m-%d")
            end_date_str = date_range[1].strftime("%Y-%m-%d")

            with st.spinner(f"Processando dados de {start_date_str} até {end_date_str}..."):
                report_data = fetch_export_report(selected_guild_id, start_date_str, end_date_str, selected_role_id)

            if not report_data or not report_data.get("records"):
                st.info(f"Nenhum registro de presença encontrado no período de {start_date_str} a {end_date_str}.")
            else:
                summary = report_data["summary"]
                records = report_data["records"]

                # BI Panel Summary metrics
                st.markdown("#### Métricas do Período")
                m1, m2, m3, m4, m5 = st.columns(5)

                rate = summary.get("attendance_rate_percent", 0.0)
                rate_color = "#34D399" if rate >= 75.0 else ("#FBBF24" if rate >= 50 else "#F87171")

                m1.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: {rate_color};">{rate:.1f}%</div>
                    <div class="metric-label">Taxa de Presença</div>
                </div>
                """, unsafe_allow_html=True)

                m2.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value">{summary.get("total_records", 0)}</div>
                    <div class="metric-label">Total Esperado</div>
                </div>
                """, unsafe_allow_html=True)

                m3.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: #34D399;">{summary.get("total_presents", 0)}</div>
                    <div class="metric-label">Presenças Válidas</div>
                </div>
                """, unsafe_allow_html=True)

                m4.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: #FBBF24;">{summary.get("total_lates", 0)}</div>
                    <div class="metric-label">Atrasos</div>
                </div>
                """, unsafe_allow_html=True)

                m5.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: #F87171;">{summary.get("total_absents", 0)}</div>
                    <div class="metric-label">Faltas</div>
                </div>
                """, unsafe_allow_html=True)

                st.markdown("<br>", unsafe_allow_html=True)

                st.markdown('<div class="card-section">', unsafe_allow_html=True)
                st.markdown("##### Registros Detalhados", unsafe_allow_html=True)

                df = pd.DataFrame(records)
                df = df.rename(columns={
                    "student_name": "Nome do Aluno",
                    "student_discord_id": "Discord ID",
                    "role_name": "Turma / Role",
                    "date": "Data",
                    "clock_in": "Entrada",
                    "clock_out": "Saída",
                    "status": "Status"
                })

                status_map = {
                    "completed": "✅ Completo",
                    "late": "⏳ Atrasado",
                    "absent": "❌ Falta",
                    "justified": "📝 Justificado",
                    "pending_checkout": "🔄 Em Progresso"
                }
                df["Status"] = df["Status"].map(lambda x: status_map.get(x, x))

                st.dataframe(df, use_container_width=True, hide_index=True)

                csv_bytes = df.to_csv(index=False).encode('utf-8')

                st.download_button(
                    label="📥 Baixar Planilha de Presenças (CSV)",
                    data=csv_bytes,
                    file_name=f"relatorio_presencas_{start_date_str}_a_{end_date_str}.csv",
                    mime="text/csv",
                    type="primary",
                    key="historical_download_btn"
                )
                st.markdown('</div>', unsafe_allow_html=True)
