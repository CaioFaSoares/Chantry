import streamlit as st
import datetime
import time
from utils.api_client import (
    fetch_guilds,
    fetch_broadcast_page_data,
    schedule_broadcast,
    cancel_broadcast,
    get_server_timezone
)

# 1. Page Config
st.set_page_config(
    page_title="Central de Megafone & Mensagens - Chantry",
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

# 3. Initialize Session State for drafts and clones
if "draft_content" not in st.session_state:
    st.session_state.draft_content = ""
if "draft_target_type" not in st.session_state:
    st.session_state.draft_target_type = "public"
if "draft_target_roles" not in st.session_state:
    st.session_state.draft_target_roles = []
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None

# Sidebar
st.sidebar.image("https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=300&q=80", width=True)
st.sidebar.markdown("<h2 style='text-align: center;'>Chantry Suite</h2>", unsafe_allow_html=True)
st.sidebar.markdown("---")

# Title
st.markdown("<h1 class='main-header'>📢 Central de Megafone & Mensagens</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>Orquestre e agende comunicados gerais ou DMs privadas direcionadas para seus alunos no Discord</p>", unsafe_allow_html=True)

# 4. Fetch Guilds
guilds = fetch_guilds()
if guilds is None:
    st.error("❌ **Erro de Conexão:** Não foi possível conectar ao Go Server Daemon na porta 12000.")
elif not guilds:
    st.warning("⚠️ **Sem Servidores:** O bot não está presente em nenhuma guilda autorizada.")
else:
    # Main Area Guild Selector
    guild_options = {g["name"]: g["id"] for g in guilds}
    selected_name = st.selectbox(
        "Selecione o Servidor",
        options=list(guild_options.keys()),
        index=0
    )
    guild_discord_id = guild_options[selected_name]
    st.session_state.selected_guild_id = guild_discord_id

    # Fetch Aggregated BFF Page Data
    page_data = fetch_broadcast_page_data(guild_discord_id)

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

        # Tab Selection Setup
        tabs = st.tabs(["📝 Compor & Agendar", "⏳ Agendados", "📜 Histórico de Disparos"])

        # =========================================================================
        # TAB 1: COMPOSITION
        # Reactive controls (destination, delivery type, roles, date/time) are
        # rendered OUTSIDE st.form so that Streamlit re-renders them immediately
        # when the user changes a value. The form only holds the message textarea
        # and the submit button.
        # =========================================================================
        with tabs[0]:
            st.markdown("<h3 class='section-title'>📝 Criar Novo Comunicado</h3>", unsafe_allow_html=True)

            dest_options = [
                "Aviso Geral (Canal de Avisos)",
                "Mensagem Direta (Canais 1-on-1) - Todos os Alunos",
                "Mensagem Direta (Canais 1-on-1) - Filtrar por Cargos"
            ]

            default_dest_idx = 0
            if st.session_state.draft_target_type == "private":
                default_dest_idx = 2 if len(st.session_state.draft_target_roles) > 0 else 1

            # --- Reactive widget 1: Destination ---
            selected_destination = st.selectbox(
                "🎯 Destino do Comunicado",
                options=dest_options,
                index=default_dest_idx,
                key="compose_destination"
            )

            # --- Reactive widget 2: Role multiselect (only when filtering by role) ---
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

            # --- Reactive widget 3: Delivery type ---
            delivery_type = st.radio(
                "⏱️ Tipo de Envio",
                options=["Enviar Agora", "Agendar Envio"],
                index=0,
                horizontal=True,
                key="compose_delivery_type"
            )

            # --- Reactive widgets 4 & 5: Date / Time (disabled when Enviar Agora) ---
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

            # --- Form: only message content + submit ---
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
                    # Read reactive state values captured before form submission
                    _destination = st.session_state.get("compose_destination", dest_options[0])
                    _delivery = st.session_state.get("compose_delivery_type", "Enviar Agora")
                    _roles = st.session_state.get("compose_roles", [])
                    _date = st.session_state.get("compose_date", datetime.date.today())
                    _time = st.session_state.get("compose_time", datetime.time(12, 0))

                    # Validation
                    if not content.strip():
                        st.error("❌ **Erro:** O conteúdo do comunicado não pode ser vazio.")
                    elif _destination == "Aviso Geral (Canal de Avisos)" and not announcement_channel_id:
                        st.error("❌ **Erro:** Canal de avisos não configurado. Acesse a aba de Infraestrutura para configurar.")
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
                            import pytz
                            # Localize the naive datetime to the server timezone before converting to UTC
                            server_tz_str = get_server_timezone()
                            local_tz = pytz.timezone(server_tz_str)
                            naive_dt = datetime.datetime.combine(_date, _time)
                            local_dt = local_tz.localize(naive_dt)
                            scheduled_dt = local_dt.astimezone(datetime.timezone.utc)

                        scheduled_str = scheduled_dt.strftime("%Y-%m-%d %H:%M:%S.000Z")

                        with st.spinner("Persistindo agendamento no banco de dados..."):
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

        # =========================================================================
        # TAB 2: SCHEDULED BROADCASTS
        # =========================================================================
        with tabs[1]:
            col_t1, col_t2 = st.columns([0.8, 0.2])
            col_t1.markdown("<h3 class='section-title'>⏳ Comunicados Agendados</h3>", unsafe_allow_html=True)
            if col_t2.button("🔄 Atualizar Lista", use_container_width=True):
                fetch_broadcast_page_data.clear()
                st.rerun()
                
            scheduled_items = [b for b in broadcasts if b.get("status") == "scheduled"]

            if not scheduled_items:
                st.info("ℹ️ **Nenhum agendamento pendente:** Todos os comunicados programados já foram disparados.")
            else:
                for item in scheduled_items:
                    try:
                        import pytz
                        # Format is YYYY-MM-DD HH:mm:ss.SSSZ
                        dt_utc = datetime.datetime.strptime(item["schedule_time"], "%Y-%m-%d %H:%M:%S.000Z")
                        dt_utc = dt_utc.replace(tzinfo=datetime.timezone.utc)
                        server_tz_str = get_server_timezone()
                        local_tz = pytz.timezone(server_tz_str)
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

        # =========================================================================
        # TAB 3: HISTORY
        # =========================================================================
        with tabs[2]:
            st.markdown("<h3 class='section-title'>📜 Histórico de Envios</h3>", unsafe_allow_html=True)
            history_items = [b for b in broadcasts if b.get("status") in ["processing", "completed", "failed"]]

            if not history_items:
                st.info("ℹ️ **Sem histórico:** Nenhuma mensagem foi enviada ou processada ainda.")
            else:
                for item in history_items:
                    status = item.get("status")
                    if status == "processing":
                        status_badge = "<span class='badge-info'>⚙️ PROCESSANDO</span>"
                    elif status == "completed":
                        status_badge = "<span class='badge-success'>✅ CONCLUÍDO</span>"
                    else:
                        status_badge = "<span class='badge-danger'>❌ FALHOU</span>"

                    dest_badge = "Megafone Público" if item["target_type"] == "public" else "Mensagem Direta 1-on-1"
                    metrics_str = f"🚀 **{item.get('metrics_sent', 0)}** enviadas | ⚠️ **{item.get('metrics_errors', 0)}** erros"

                    with st.container():
                        st.markdown(
                            f"<div class='card-section' style='margin-bottom: 16px;'>"
                            f"<div style='display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;'>"
                            f"{status_badge}"
                            f"<span style='color: #64748B; font-size: 0.85rem;'>Destino: <strong>{dest_badge}</strong></span>"
                            f"</div>"
                            f"<div style='color: #94A3B8; font-size: 0.9rem; margin-bottom: 12px;'>Métricas: {metrics_str}</div>"
                            f"<p style='color: #CBD5E1; white-space: pre-wrap; font-size: 0.9rem; background: rgba(0, 0, 0, 0.15); padding: 12px; border-radius: 8px;'>{item['content']}</p>"
                            f"</div>",
                            unsafe_allow_html=True
                        )

                        if st.button("🔄 Clonar & Editar Conteúdo", key=f"clone_{item['id']}", use_container_width=True):
                            st.session_state.draft_content = item["content"]
                            st.session_state.draft_target_type = item["target_type"]
                            st.session_state.draft_target_roles = item.get("target_roles", [])
                            st.success("📝 **Comunicado Clonado!** O conteúdo foi carregado na aba 'Compor & Agendar'.")
                            time.sleep(1.0)
                            st.rerun()
