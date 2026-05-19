import streamlit as st
import datetime
import time

# 1. Page Config
st.set_page_config(
    page_title="Hub de Engajamento - Chantry",
    page_icon="📢",
    layout="wide",
    initial_sidebar_state="expanded"
)

# 2. Premium CSS (Outfit font, sleek dark mode cards, glowing highlights)
custom_css = """
<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap');

html, body, [class*="css"], .stMarkdown, .stButton button {
    font-family: 'Outfit', sans-serif !important;
}

.main-header {
    background: linear-gradient(135deg, #FF6B6B 0%, #FF8E53 50%, #FFD200 100%);
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
    background: rgba(30, 41, 59, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 16px;
    padding: 24px;
    backdrop-filter: blur(12px);
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.2);
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

.badge-info {
    display: inline-block;
    padding: 4px 8px;
    background: rgba(59, 130, 246, 0.15);
    border: 1px solid rgba(59, 130, 246, 0.3);
    border-radius: 8px;
    color: #60A5FA;
    font-size: 0.85rem;
    font-weight: 500;
}

.badge-success {
    display: inline-block;
    padding: 4px 8px;
    background: rgba(16, 185, 129, 0.15);
    border: 1px solid rgba(16, 185, 129, 0.3);
    border-radius: 8px;
    color: #34D399;
    font-size: 0.85rem;
    font-weight: 500;
}

.badge-warning {
    display: inline-block;
    padding: 4px 8px;
    background: rgba(245, 158, 11, 0.15);
    border: 1px solid rgba(245, 158, 11, 0.3);
    border-radius: 8px;
    color: #FBBF24;
    font-size: 0.85rem;
    font-weight: 500;
}

.badge-danger {
    display: inline-block;
    padding: 4px 8px;
    background: rgba(239, 68, 68, 0.15);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: 8px;
    color: #F87171;
    font-size: 0.85rem;
    font-weight: 500;
}
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

from utils.api_client import (
    fetch_guilds,
    fetch_broadcast_page_data,
    schedule_broadcast,
    cancel_broadcast,
    get_server_timezone
)

# Initialize Session State for drafts and clones
if "draft_content" not in st.session_state:
    st.session_state.draft_content = ""
if "draft_target_type" not in st.session_state:
    st.session_state.draft_target_type = "public"
if "draft_target_roles" not in st.session_state:
    st.session_state.draft_target_roles = []
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None

# Title
st.markdown("<h1 class='main-header'>📢 Hub de Engajamento</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>Comunicação, Megafone, mensageria direta e calendário da sua guilda Discord</p>", unsafe_allow_html=True)

# Fetch Guilds list
guilds = fetch_guilds()
if guilds is None:
    st.error("❌ **Erro de Conexão:** Não foi possível conectar ao Go Server Daemon na porta 12000.")
    st.stop()

if not guilds:
    st.warning("⚠️ **Sem Servidores:** O bot não está presente em nenhuma guilda autorizada.")
    st.stop()

# 3. Context Selector in Main Area
st.markdown("<div class='card-section'>", unsafe_allow_html=True)
st.markdown("<div class='section-title'>🔌 Seleção do Servidor de Engajamento</div>", unsafe_allow_html=True)

guild_options = {g["id"]: g["name"] for g in guilds}
default_index = 0
if st.session_state.selected_guild_id and st.session_state.selected_guild_id in guild_options:
    default_index = list(guild_options.keys()).index(st.session_state.selected_guild_id)

selected_guild_id = st.selectbox(
    "Escolha o Servidor Discord para engajamento:",
    options=list(guild_options.keys()),
    index=default_index,
    format_func=lambda x: guild_options[x],
    key="global_engagement_guild_selector"
)
st.session_state.selected_guild_id = selected_guild_id

st.markdown("</div>", unsafe_allow_html=True)

# Sidebar
st.sidebar.markdown("### 🛠️ Controles de Atualização")
if st.sidebar.button("🔄 Atualizar Hub", use_container_width=True):
    st.rerun()

# Set up Tabbed Interface
tab_broadcast, tab_calendar = st.tabs([
    "📣 Central de Mensagens (Megafone)", 
    "📅 Calendário de Aulas"
])

# ==========================================
# TAB 1: Central de Mensagens (Megafone)
# ==========================================
with tab_broadcast:
    # Fetch Aggregated BFF Page Data
    page_data = fetch_broadcast_page_data(selected_guild_id)

    if not page_data:
        st.error("❌ **Erro de Dados:** Não foi possível carregar as informações desta guilda.")
    else:
        announcement_channel_id = page_data.get("announcement_channel_id", "")
        announcement_channel_name = page_data.get("announcement_channel_name", "Não Configurado")
        roles = page_data.get("roles", [])
        broadcasts = page_data.get("broadcasts", [])

        # UI Layout: Configured channel status bar
        status_color = "green" if announcement_channel_id else "orange"
        st.markdown(
            f"<div class='card-section'>"
            f"<span style='color: {status_color}; font-size: 1.1rem; font-weight: 600;'>Mega Megafone:</span> "
            f"<span class='badge-info'>{announcement_channel_name}</span> "
            f"<span style='color: #64748B; font-size: 0.9rem; margin-left: 8px;'>ID: {announcement_channel_id or 'Pendente'}</span>"
            f"</div>",
            unsafe_allow_html=True
        )

        # Sub-navigation using st.radio to avoid nested tabs styling issues
        sub_tab = st.radio(
            "Selecione a Ação da Central de Mensagens:",
            options=["📝 Compor & Agendar", "⏳ Agendados", "📜 Histórico de Disparos"],
            horizontal=True,
            key="engagement_sub_navigation"
        )

        # ----------------------------------------------------
        # SUB-TAB 1: Compor & Agendar
        # ----------------------------------------------------
        if sub_tab == "📝 Compor & Agendar":
            st.markdown("<h3 class='section-title'>📝 Criar Novo Comunicado</h3>", unsafe_allow_html=True)

            dest_options = [
                "Aviso Geral (Canal de Avisos)",
                "Mensagem Direta (Canais 1-on-1) - Todos os Alunos",
                "Mensagem Direta (Canais 1-on-1) - Filtrar por Cargos"
            ]

            default_dest_idx = 0
            if st.session_state.draft_target_type == "private":
                default_dest_idx = 2 if len(st.session_state.draft_target_roles) > 0 else 1

            # Destination
            selected_destination = st.selectbox(
                "🎯 Destino do Comunicado",
                options=dest_options,
                index=default_dest_idx,
                key="compose_destination"
            )

            # Role multiselect
            role_options = {r["name"]: r["id"] for r in roles}
            selected_role_names = []
            if selected_destination == "Mensagem Direta (Canais 1-on-1) - Filtrar por Cargos":
                preselected_roles = [
                    name for name, rid in role_options.items()
                    if rid in st.session_state.draft_target_roles
                ]
                selected_role_names = st.multiselect(
                    "🏷️ Selecione os Cargos Alvos",
                    options=list(role_options.keys()),
                    default=preselected_roles,
                    key="compose_roles"
                )

            st.markdown("---")

            # Delivery type
            delivery_type = st.radio(
                "⏱️ Tipo de Envio",
                options=["Enviar Agora", "Agendar Envio"],
                index=0,
                horizontal=True,
                key="compose_delivery_type"
            )

            is_immediate = (delivery_type == "Enviar Agora")
            col_date, col_time = st.columns(2)
            with col_date:
                min_date = datetime.date.today()
                sched_date = st.date_input(
                    "📅 Data do Envio",
                    value=min_date,
                    min_value=min_date,
                    disabled=is_immediate,
                    key="compose_date"
                )
            with col_time:
                sched_time = st.time_input(
                    "🕐 Horário do Envio",
                    value=datetime.time(12, 0),
                    disabled=is_immediate,
                    key="compose_time"
                )

            # Form
            with st.form("broadcast_form", clear_on_submit=True):
                content = st.text_area(
                    "✉️ Conteúdo da Mensagem",
                    value=st.session_state.draft_content,
                    height=200,
                    placeholder="Escreva a mensagem usando markdown, menções ou emojis..."
                )

                submit_label = "🚀 Disparar Comunicado" if is_immediate else "⏰ Agendar Disparo"
                submitted = st.form_submit_button(submit_label, use_container_width=True, type="primary")

                if submitted:
                    _destination = st.session_state.get("compose_destination", dest_options[0])
                    _delivery = st.session_state.get("compose_delivery_type", "Enviar Agora")
                    _roles = st.session_state.get("compose_roles", [])
                    _date = st.session_state.get("compose_date", datetime.date.today())
                    _time = st.session_state.get("compose_time", datetime.time(12, 0))

                    if not content.strip():
                        st.error("❌ **Erro:** O conteúdo do comunicado não pode ser vazio.")
                    elif _destination == "Aviso Geral (Canal de Avisos)" and not announcement_channel_id:
                        st.error("❌ **Erro:** Canal de avisos não configurado na aba de Infraestrutura (Setup).")
                    elif _destination == "Mensagem Direta (Canais 1-on-1) - Filtrar por Cargos" and not _roles:
                        st.error("❌ **Erro:** Selecione ao menos um cargo alvo para entrega filtrada.")
                    else:
                        target_type = "public" if "Aviso Geral" in _destination else "private"
                        target_roles = []
                        if _destination == "Mensagem Direta (Canais 1-on-1) - Filtrar por Cargos":
                            target_roles = [role_options[name] for name in _roles]

                        if _delivery == "Enviar Agora":
                            scheduled_dt = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(seconds=10)
                        else:
                            from zoneinfo import ZoneInfo
                            server_tz_str = get_server_timezone()
                            local_tz = ZoneInfo(server_tz_str)
                            naive_dt = datetime.datetime.combine(_date, _time)
                            local_dt = naive_dt.replace(tzinfo=local_tz)
                            scheduled_dt = local_dt.astimezone(datetime.timezone.utc)

                        scheduled_str = scheduled_dt.strftime("%Y-%m-%d %H:%M:%S.000Z")

                        with st.spinner("Persistindo agendamento no banco..."):
                            success, result = schedule_broadcast(
                                guild_id=page_data.get("guild_pb_id", ""),
                                content=content,
                                target_type=target_type,
                                target_roles=target_roles,
                                schedule_time=scheduled_str
                            )

                            if success:
                                st.success("✅ **Agendamento Concluído!** O comunicado foi salvo no banco e será entregue.")
                                st.session_state.draft_content = ""
                                st.session_state.draft_target_roles = []
                                time.sleep(1.5)
                                st.rerun()
                            else:
                                st.error(f"❌ **Erro no Backend:** {result}")

        # ----------------------------------------------------
        # SUB-TAB 2: Agendados
        # ----------------------------------------------------
        elif sub_tab == "⏳ Agendados":
            col_t1, col_t2 = st.columns([0.8, 0.2])
            col_t1.markdown("<h3 class='section-title'>⏳ Comunicados Agendados</h3>", unsafe_allow_html=True)
            if col_t2.button("🔄 Atualizar Lista", use_container_width=True):
                st.rerun()

            scheduled_items = [b for b in broadcasts if b.get("status") == "scheduled"]

            if not scheduled_items:
                st.info("ℹ️ **Nenhum agendamento pendente:** Todos os comunicados programados já foram disparados.")
            else:
                for item in scheduled_items:
                    try:
                        from zoneinfo import ZoneInfo
                        dt_utc = datetime.datetime.strptime(item["schedule_time"], "%Y-%m-%d %H:%M:%S.000Z")
                        dt_utc = dt_utc.replace(tzinfo=datetime.timezone.utc)
                        server_tz_str = get_server_timezone()
                        local_tz = ZoneInfo(server_tz_str)
                        dt_local = dt_utc.astimezone(local_tz)
                        local_time_str = dt_local.strftime("%d/%m/%Y às %H:%M")
                    except Exception:
                        local_time_str = item["schedule_time"]

                    dest_badge = "Megafone Público" if item["target_type"] == "public" else "Mensagem Direta 1-on-1"

                    with st.container():
                        st.markdown(
                            f"<div class='card-section' style='border-left: 5px solid #F59E0B; margin-bottom: 16px;'>"
                            f"<div style='display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;'>"
                            f"<span class='badge-warning'>⏳ AGENDADO</span>"
                            f"<span style='color: #94A3B8; font-size: 0.9rem;'>Disparo previsto: <strong>{local_time_str}</strong></span>"
                            f"</div>"
                            f"<div style='color: #64748B; font-size: 0.85rem; margin-bottom: 12px;'>Destino: <strong>{dest_badge}</strong></div>"
                            f"<p style='color: #F1F5F9; white-space: pre-wrap; font-size: 0.95rem; background: rgba(0, 0, 0, 0.2); padding: 12px; border-radius: 8px;'>{item['content']}</p>"
                            f"</div>",
                            unsafe_allow_html=True
                        )

                        if st.button("❌ Cancelar & Excluir Agendamento", key=f"cancel_{item['id']}", use_container_width=True):
                            with st.spinner("Excluindo agendamento..."):
                                success, message = cancel_broadcast(item["id"])
                                if success:
                                    st.success("✅ Agendamento excluído!")
                                    time.sleep(1.0)
                                    st.rerun()
                                else:
                                    st.error(f"Erro ao cancelar: {message}")

        # ----------------------------------------------------
        # SUB-TAB 3: Histórico de Disparos
        # ----------------------------------------------------
        elif sub_tab == "📜 Histórico de Disparos":
            st.markdown("<h3 class='section-title'>📜 Histórico de Envios</h3>", unsafe_allow_html=True)
            history_items = [b for b in broadcasts if b.get("status") in ["processing", "completed", "failed"]]

            if not history_items:
                st.info("ℹ️ **Sem histórico:** Nenhuma mensagem foi enviada ou processada ainda.")
            else:
                role_id_to_name = {r["id"]: r["name"] for r in roles}

                for item in history_items:
                    status = item.get("status")
                    if status == "processing":
                        status_badge = "<span class='badge-info'>⚙️ PROCESSANDO</span>"
                    elif status == "completed":
                        status_badge = "<span class='badge-success'>✅ CONCLUÍDO</span>"
                    else:
                        status_badge = "<span class='badge-danger'>❌ FALHOU</span>"

                    try:
                        from zoneinfo import ZoneInfo
                        dt_utc = datetime.datetime.strptime(item["schedule_time"], "%Y-%m-%d %H:%M:%S.000Z")
                        dt_utc = dt_utc.replace(tzinfo=datetime.timezone.utc)
                        server_tz_str = get_server_timezone()
                        local_tz = ZoneInfo(server_tz_str)
                        dt_local = dt_utc.astimezone(local_tz)
                        local_time_str = dt_local.strftime("%d/%m/%Y às %H:%M:%S")
                    except Exception:
                        local_time_str = item.get("schedule_time", "Data Desconhecida")

                    if item.get("target_type") == "public":
                        dest_badge = "📢 Megafone Público (Aviso Geral)"
                    else:
                        t_roles = item.get("target_roles", [])
                        if t_roles and len(t_roles) > 0:
                            role_names = [role_id_to_name.get(rid, "Desconhecido") for rid in t_roles]
                            dest_badge = f"🎯 DM Privada (Turmas: {', '.join(role_names)})"
                        else:
                            dest_badge = "🎯 DM Privada (Todos os Alunos)"

                    metrics_str = f"🚀 **{item.get('metrics_sent', 0)}** enviadas | ⚠️ **{item.get('metrics_errors', 0)}** erros"

                    with st.container():
                        st.markdown(
                            f"<div class='card-section' style='margin-bottom: 16px;'>"
                            f"<div style='display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;'>"
                            f"<div>{status_badge} <span style='color: #94A3B8; font-size: 0.85rem; margin-left: 8px;'>Disparado: <strong>{local_time_str}</strong></span></div>"
                            f"</div>"
                            f"<div style='color: #64748B; font-size: 0.9rem; margin-bottom: 4px;'>Alvo: <strong>{dest_badge}</strong></div>"
                            f"<div style='color: #94A3B8; font-size: 0.9rem; margin-bottom: 12px;'>Métricas: {metrics_str}</div>"
                            f"<p style='color: #CBD5E1; white-space: pre-wrap; font-size: 0.9rem; background: rgba(0, 0, 0, 0.15); padding: 12px; border-radius: 8px;'>{item['content']}</p>"
                            f"</div>",
                            unsafe_allow_html=True
                        )

                        if st.button("🔄 Clonar & Editar Conteúdo", key=f"clone_{item['id']}", use_container_width=True):
                            st.session_state.draft_content = item["content"]
                            st.session_state.draft_target_type = item["target_type"]
                            st.session_state.draft_target_roles = item.get("target_roles", [])
                            st.success("📝 **Comunicado Clonado!** O conteúdo foi carregado na ação 'Compor & Agendar'.")
                            time.sleep(1.0)
                            st.rerun()


# ==========================================
# TAB 2: Calendário de Aulas (Placeholder)
# ==========================================
with tab_calendar:
    st.markdown("### 📅 Calendário de Aulas e Mentorias")
    st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top:-10px; margin-bottom: 20px;'>"
                "Gerenciamento de eventos escolares integrados ao Discord.</p>",
                unsafe_allow_html=True)
    
    st.info("🚧 **Em construção:** O Motor de Calendário e Encontros chegará em breve! "
            "Aqui você poderá agendar aulas e mentorias, notificando turmas ou alunos específicos diretamente no Discord.")
