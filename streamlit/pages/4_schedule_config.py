import streamlit as st
import requests
import socket
import time
from datetime import datetime

# 1. Page Config
st.set_page_config(
    page_title="Configuração de Turnos - Chantry",
    page_icon="📅",
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
    background: linear-gradient(135deg, #F59E0B 0%, #EC4899 50%, #8B5CF6 100%);
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
    background: linear-gradient(135deg, rgba(245, 158, 11, 0.15) 0%, rgba(236, 72, 153, 0.15) 100%);
    border: 1px solid rgba(245, 158, 11, 0.3);
    color: #FBBF24;
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
    fetch_guilds,
    fetch_guild_roles_config,
    update_role_config,
    fetch_guild_managers,
    trigger_test_attendance
)

# Initialize global session state
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None
st.markdown('<h1 class="main-header">📅 Configuração de Turnos</h1>', unsafe_allow_html=True)
st.markdown('<p class="subtitle">Gerencie o monitoramento de presença e automatize os horários de ponto da sua comunidade</p>', unsafe_allow_html=True)

# 5. TZ Server Time Monitor
health = fetch_server_health()
if health and "timestamp" in health:
    try:
        dt_str = health["timestamp"]
        dt = datetime.fromisoformat(dt_str)
        formatted_time = dt.strftime("%H:%M:%S (%d/%m/%Y)")
        tz_name = health.get("timezone", "America/Sao_Paulo")
        
        st.markdown(
            f'<div class="clock-badge">⏰ <b>Horário do Servidor:</b> {formatted_time} &nbsp;|&nbsp; 🌍 <b>Timezone:</b> {tz_name}</div>',
            unsafe_allow_html=True
        )
    except Exception:
        st.markdown(
            f'<div class="clock-badge">⏰ <b>Horário do Servidor:</b> {health["timestamp"]} &nbsp;|&nbsp; 🌍 <b>Timezone:</b> America/Sao_Paulo</div>',
            unsafe_allow_html=True
        )
else:
    st.markdown(
        '<div class="clock-badge" style="color: #EF4444; border-color: rgba(239, 68, 68, 0.3); background: rgba(239, 68, 68, 0.1);">⚠️ <b>Status do Daemon Go:</b> Offline ou inacessível</div>',
        unsafe_allow_html=True
    )

# 6. Sidebar controls
st.sidebar.markdown("### 🛠️ Configurações Globais")
refresh_btn = st.sidebar.button("🔄 Atualizar Relógio do Servidor")
if refresh_btn:
    st.rerun()

# 7. Load Guilds
guilds = fetch_guilds()

if guilds is None:
    st.error("❌ Não foi possível se conectar ao Chantry Go Daemon. Por favor, verifique se a API está online.")
elif not guilds:
    st.warning("⚠️ Nenhum servidor de Discord (Guild) encontrado ou mapeado. Conecte o bot a um servidor.")
else:
    # Select Server
    st.markdown('<div class="card-section">', unsafe_allow_html=True)
    st.markdown('<div class="section-title">🔌 Passo 1: Selecionar Servidor Alvo</div>', unsafe_allow_html=True)
    
    guild_options = {g["id"]: g["name"] for g in guilds}
    
    default_index = 0
    if st.session_state.selected_guild_id and st.session_state.selected_guild_id in guild_options:
        default_index = list(guild_options.keys()).index(st.session_state.selected_guild_id)

    selected_guild_id = st.selectbox(
        "Selecione o Servidor (Guild)",
        options=list(guild_options.keys()),
        index=default_index,
        format_func=lambda gid: guild_options[gid]
    )
    st.session_state.selected_guild_id = selected_guild_id
    st.markdown('</div>', unsafe_allow_html=True)

    if selected_guild_id:
        with st.spinner("Buscando cargos e configurações da guilda..."):
            roles = fetch_guild_roles_config(selected_guild_id)
        
        if not roles:
            st.info("ℹ️ Nenhuma role/turma foi sincronizada ou encontrada neste servidor. Vá até a página de Sincronização do Discord primeiro.")
        else:
            # 8. Render Tabs
            tab1, tab2 = st.tabs(["⏰ Horários de Ponto", "⚙️ Triagem de Turmas"])

            # ==================== TAB 1: Horários de Ponto ====================
            with tab1:
                monitored_roles = [r for r in roles if r.get("is_monitored", False)]

                if not monitored_roles:
                    st.info("ℹ️ Nenhuma turma foi selecionada para monitoramento ainda. Acesse a aba '⚙️ Triagem de Turmas' acima para habilitar o controle de ponto em seus cargos.")
                else:
                    st.markdown('<h3 style="font-size: 1.3rem; font-weight: 600; color: #F1F5F9; margin-bottom: 16px;">📚 Horários das Turmas Monitoradas</h3>', unsafe_allow_html=True)
                    
                    shift_mapping = {
                        "morning": "Manhã",
                        "afternoon": "Tarde",
                        "night": "Noite"
                    }
                    reverse_shift_mapping = {v: k for k, v in shift_mapping.items()}

                    for index, role in enumerate(monitored_roles):
                        role_pb_id = role["id"]
                        role_discord_name = role["name"]
                        
                        current_shift_db = role.get("shift", "")
                        current_shift_lbl = shift_mapping.get(current_shift_db, "Manhã")
                        
                        current_check_in = role.get("check_in_time", "")
                        if current_check_in == "":
                            current_check_in = "08:00"
                        
                        current_cooldown = role.get("checkout_cooldown", 0)
                        if current_cooldown <= 0:
                            current_cooldown = 4
                            
                        current_is_active = role.get("is_active", False)

                        # Render card
                        st.markdown(f'<div class="card-section">', unsafe_allow_html=True)
                        
                        # Title row with Active status toggle
                        tcol1, tcol2 = st.columns([3, 1])
                        with tcol1:
                            st.markdown(f'<div class="section-title" style="margin-bottom:0px;">🎓 Turma: <b>{role_discord_name}</b> <span style="font-size:0.85rem; font-weight:normal; color:#64748B;">({role_pb_id})</span></div>', unsafe_allow_html=True)
                        with tcol2:
                            val_active = st.toggle(
                                "Agendamento Ativo",
                                value=current_is_active,
                                key=f"active_{role_pb_id}_{index}"
                            )

                        st.markdown('<div style="height:15px;"></div>', unsafe_allow_html=True)
                        
                        # 3 columns for shift, check-in, cooldown
                        col1, col2, col3 = st.columns(3)
                        with col1:
                            sel_shift_lbl = st.selectbox(
                                "Turno de Estudo",
                                options=["Manhã", "Tarde", "Noite"],
                                index=["Manhã", "Tarde", "Noite"].index(current_shift_lbl),
                                key=f"shift_{role_pb_id}_{index}"
                            )
                        with col2:
                            val_check_in = st.text_input(
                                "Horário de Entrada (Disparo)",
                                value=current_check_in,
                                placeholder="HH:MM",
                                help="Formato de 24 horas (ex: 08:00 ou 14:30)",
                                key=f"checkin_{role_pb_id}_{index}"
                            )
                        with col3:
                            val_cooldown = st.number_input(
                                "Janela de Saída em Horas (Check-Out)",
                                min_value=1,
                                max_value=12,
                                value=current_cooldown,
                                help="Duração em horas em que o check-out do aluno é liberado",
                                key=f"cooldown_{role_pb_id}_{index}"
                            )

                        st.write("")
                        col_btn, _ = st.columns([1, 4])
                        with col_btn:
                            save_btn = st.button(
                                f"💾 Salvar Configurações",
                                key=f"save_{role_pb_id}_{index}",
                                use_container_width=True
                            )

                        if save_btn:
                            import re
                            if not re.match(r"^(0[0-9]|1[0-9]|2[0-3]):[0-5][0-9]$", val_check_in):
                                st.error(f"❌ Horário de entrada inválido: {val_check_in}. Deve ser HH:MM (ex: 08:00)")
                            else:
                                shift_db_val = reverse_shift_mapping[sel_shift_lbl]
                                success, result = update_role_config(
                                    role_pb_id,
                                    shift=shift_db_val,
                                    check_in_time=val_check_in,
                                    checkout_cooldown=int(val_cooldown),
                                    is_active=val_active
                                )
                                if success:
                                    st.success(f"✅ Configurações de {role_discord_name} salvas com sucesso!")
                                    time.sleep(0.5)
                                    st.rerun()
                                else:
                                    st.error(f"❌ Erro ao salvar configurações: {result}")
                        
                        st.markdown('</div>', unsafe_allow_html=True)

            # ==================== TAB 2: Triagem de Turmas ====================
            with tab2:
                st.markdown('<h3 style="font-size: 1.3rem; font-weight: 600; color: #F1F5F9; margin-bottom: 12px;">⚙️ Triagem de Cargos do Discord</h3>', unsafe_allow_html=True)
                st.markdown('<p style="color: #94A3B8; font-size: 0.95rem; margin-bottom: 20px;">Defina quais cargos representam de fato Turmas/Squads de alunos que participarão do controle de presença diário. Cargos secundários (como mentores, administradores, habilidades específicas) devem ficar de fora.</p>', unsafe_allow_html=True)

                st.markdown('<div class="card-section">', unsafe_allow_html=True)
                
                # Fetch currently selected monitored IDs
                current_monitored_ids = [r["id"] for r in roles if r.get("is_monitored", False)]
                
                selected_role_ids = st.multiselect(
                    "Selecione os cargos/turmas monitorados:",
                    options=[r["id"] for r in roles],
                    default=current_monitored_ids,
                    format_func=lambda rid: next((r["name"] for r in roles if r["id"] == rid), rid),
                    help="Apenas as selecionadas aparecerão na aba de Horários de Ponto.",
                    key="triagem_multiselect"
                )

                st.write("")
                col_save_triagem, _ = st.columns([1, 3])
                with col_save_triagem:
                    save_triagem_btn = st.button("💾 Salvar Triagem de Turmas", use_container_width=True)

                if save_triagem_btn:
                    success_count = 0
                    errors = []
                    
                    with st.spinner("Atualizando triagem de turmas..."):
                        for r in roles:
                            rid = r["id"]
                            should_be_monitored = rid in selected_role_ids
                            was_monitored = r.get("is_monitored", False)
                            
                            # Only issue requests for changes
                            if should_be_monitored != was_monitored:
                                payload = {"is_monitored": should_be_monitored}
                                # Automatically activate when enabling monitoring for the first time
                                if should_be_monitored and not r.get("is_active", False):
                                    payload["is_active"] = True
                                
                                success, err_msg = update_role_config(rid, **payload)
                                if success:
                                    success_count += 1
                                else:
                                    errors.append(f"{r['name']}: {err_msg}")
                    
                    if errors:
                        st.error(f"❌ Ocorreram erros ao salvar algumas turmas:\n" + "\n".join(errors))
                    
                    if success_count > 0 or not errors:
                        st.success(f"✅ Triagem de turmas atualizada! {success_count} cargo(s) modificado(s).")
                        time.sleep(0.5)
                        st.rerun()

                st.markdown('</div>', unsafe_allow_html=True)

            # ==================== SANDBOX DRY RUN ====================
            st.markdown('<br>', unsafe_allow_html=True)
            with st.expander("🧪 Sandbox de Testes (Dry Run)", expanded=False):
                st.markdown(
                    '<p style="color: #94A3B8; font-size: 0.95rem; margin-bottom: 16px;">'
                    'Use esta ferramenta para simular o envio da mensagem oficial de ponto com os botões interativos '
                    'de Check-In/Check-Out em um canal de texto arbitrário do Discord. O bot tratará o usuário '
                    'selecionado como um aluno tester, garantindo integridade de banco sem poluir os relatórios escolares ativos.'
                    '</p>',
                    unsafe_allow_html=True
                )

                # Fetch managers associated with this guild
                managers = fetch_guild_managers(selected_guild_id)
                if not managers:
                    st.warning("⚠️ Nenhum manager/mentor encontrado cadastrado para esta guilda. Sincronize os usuários primeiro.")
                else:
                    manager_options = {m["discord_id"]: f"{m['name']} ({m.get('role', 'mentor')})" for m in managers}
                    
                    mcol1, mcol2 = st.columns(2)
                    with mcol1:
                        selected_tester_id = st.selectbox(
                            "Quem vai testar o botão? (Equipe/Manager)",
                            options=list(manager_options.keys()),
                            format_func=lambda mid: manager_options[mid]
                        )
                    with mcol2:
                        test_channel_id = st.text_input(
                            "ID do Canal de Teste no Discord",
                            placeholder="Ex: 112233445566778899",
                            help="Cole o ID numérico do canal do Discord onde a mensagem deve ser enviada."
                        )

                    st.write("")
                    col_trigger, _ = st.columns([1, 3])
                    with col_trigger:
                        trigger_btn = st.button("🚀 Disparar Teste de Ponto", use_container_width=True, type="secondary")

                    if trigger_btn:
                        if not test_channel_id.strip():
                            st.error("❌ Por favor, informe um ID de Canal do Discord válido.")
                        else:
                            with st.spinner("Disparando mensagem de ponto para o canal..."):
                                success, msg = trigger_test_attendance(
                                    selected_guild_id,
                                    test_channel_id.strip(),
                                    selected_tester_id
                                )
                                if success:
                                    st.success(f"✅ Dry run acionado! {msg}")
                                else:
                                    st.error(f"❌ Erro ao disparar teste: {msg}")
