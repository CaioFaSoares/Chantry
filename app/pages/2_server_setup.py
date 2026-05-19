import streamlit as st
import pandas as pd
import time
from datetime import datetime

# Page config
st.set_page_config(
    page_title="Configuração do Servidor - Chantry",
    page_icon="⚙️",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Custom Premium CSS (Outfit font, linear gradients, glassmorphism)
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
"""
st.markdown(custom_css, unsafe_allow_html=True)

# API client imports
from utils.api_client import (
    fetch_server_health,
    get_server_timezone,
    fetch_guilds,
    fetch_roles,
    fetch_members,
    sync_advanced_to_db,
    fetch_guild_roles_config,
    fetch_squad_dashboard_data,
    update_squad_channel,
    update_role_config,
    fetch_guild_managers,
    trigger_test_attendance,
    fetch_categories,
    create_category,
    provision_channels,
    heal_channels,
    get_provision_page_data,
    save_announcement_channel
)

# Sidebar
st.sidebar.image("https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=300&q=80", use_container_width=True)
st.sidebar.markdown("<h2 style='text-align: center;'>Chantry Suite</h2>", unsafe_allow_html=True)
st.sidebar.markdown("---")
st.sidebar.info("⚙️ **Configuração Unificada:** Use esta tela para configurar completamente novos servidores do Discord, sincronizar membros e estruturar salas.")

# Header
st.markdown("<h1 class='main-header'>⚙️ Configuração do Servidor</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>Gerenciamento centralizado de sincronização, turmas, regras de ponto e provisionamento</p>", unsafe_allow_html=True)

# Initialize global session states
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None
if "fetched_members" not in st.session_state:
    st.session_state.fetched_members = None
if "sync_metrics" not in st.session_state:
    st.session_state.sync_metrics = None
if "prev_guild_id" not in st.session_state:
    st.session_state.prev_guild_id = None
if "prev_role_id" not in st.session_state:
    st.session_state.prev_role_id = None
if "auto_select_category_id" not in st.session_state:
    st.session_state.auto_select_category_id = None
if "provision_metrics" not in st.session_state:
    st.session_state.provision_metrics = None
if "heal_metrics" not in st.session_state:
    st.session_state.heal_metrics = None

# 1. Fetch Guilds list
guilds = fetch_guilds()

if guilds is None:
    st.error(
        "❌ **Erro de Conexão:** Não foi possível conectar ao Daemon do Go Server. "
        "Verifique se o serviço `go-server` está ativo na porta `12000` do Docker Compose."
    )
    st.stop()

if not guilds:
    st.warning(
        "⚠️ **Nenhum Servidor Identificado:** O bot do Chantry não foi adicionado a nenhum "
        "servidor ou o token não possui permissões adequadas de leitura."
    )
    st.stop()

# 2. Render Seletor Contexto Global
st.markdown("<div class='card-section'>", unsafe_allow_html=True)
st.markdown("<div class='section-title'>🔌 Seleção do Servidor Principal</div>", unsafe_allow_html=True)

default_index = 0
if st.session_state.selected_guild_id and any(g["id"] == st.session_state.selected_guild_id for g in guilds):
    default_index = next(i for i, g in enumerate(guilds) if g["id"] == st.session_state.selected_guild_id)

selected_guild = st.selectbox(
    "Escolha o Servidor Discord para Configuração:",
    options=guilds,
    index=default_index,
    format_func=lambda g: g["name"],
    key="global_guild_selector"
)

if not selected_guild:
    st.stop()

guild_id = selected_guild["id"]
st.session_state.selected_guild_id = guild_id

# Cascade state reset on guild switch
if st.session_state.prev_guild_id != guild_id:
    st.session_state.prev_guild_id = guild_id
    st.session_state.fetched_members = None
    st.session_state.sync_metrics = None
    st.session_state.provision_metrics = None
    st.session_state.heal_metrics = None

st.markdown("</div>", unsafe_allow_html=True)

# 3. Load Guild Data & Check Fail-Safe Conditions
with st.spinner("Carregando dados de configuração da guilda..."):
    # Load provision page aggregate data (which now includes total_students!)
    provision_data = get_provision_page_data(guild_id)
    roles_config = fetch_guild_roles_config(guild_id)

# Extract counts and data
total_students_synced = 0
if provision_data and "metrics" in provision_data:
    total_students_synced = provision_data["metrics"].get("total_students", 0)

# Filter monitored roles
monitored_roles = [r for r in roles_config if r.get("is_monitored", False)] if roles_config else []

# Render Tabs
tab1, tab2, tab3, tab4 = st.tabs([
    "1. 🔄 Sincronização", 
    "2. 🏗️ Infraestrutura",
    "3. ⏰ Regras de Ponto", 
    "4. 👥 Estrutura e Squads"
])

# ==========================================
# ABA 1: Sincronização Discord
# ==========================================
with tab1:
    st.markdown("### 🔄 Mapeamento e Sincronização de Membros")
    st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top:-10px; margin-bottom: 20px;'>"
                "Importe os usuários do Discord para a base local do PocketBase associando os cargos de turmas e equipe.</p>",
                unsafe_allow_html=True)

    st.markdown("<div class='card-section'>", unsafe_allow_html=True)
    st.markdown("<div class='section-title'>⚙️ Configurações da Sincronização Avançada</div>", unsafe_allow_html=True)

    col_sync1, col_sync2 = st.columns(2)

    roles = fetch_roles(guild_id)

    with col_sync1:
        if roles is None:
            st.error("❌ Erro ao buscar os cargos deste servidor.")
            selected_role = None
        elif not roles:
            st.warning("⚠️ Nenhum cargo personalizado encontrado no Discord.")
            selected_role = None
        else:
            selected_role = st.selectbox(
                "Cargo Primário de Alunos (Squad/Turma)",
                options=roles,
                format_func=lambda r: r["name"],
                key="sync_primary_role_select"
            )

    with col_sync2:
        selected_secondary_roles = []
        if roles:
            selected_secondary_roles = st.multiselect(
                "Cargos Secundários de Alunos (Skills / Trilhas)",
                options=roles,
                format_func=lambda r: r["name"],
                key="sync_secondary_roles_select"
            )

    if roles:
        st.markdown("<div style='margin-top: 12px;'></div>", unsafe_allow_html=True)
        with st.expander("⚙️ Mapeamento de Equipe / Managers"):
            st.markdown("<p style='color: #94A3B8; font-size: 0.9rem;'>Associe cargos do Discord aos níveis de acesso de equipe no PocketBase:</p>", unsafe_allow_html=True)
            selected_admin_roles = st.multiselect(
                "Cargos de Administradores (Admin)",
                options=roles,
                format_func=lambda r: r["name"],
                key="sync_admin_roles_select"
            )
            selected_mentor_roles = st.multiselect(
                "Cargos de Mentores (Mentor)",
                options=roles,
                format_func=lambda r: r["name"],
                key="sync_mentor_roles_select"
            )
            selected_pedagogy_roles = st.multiselect(
                "Cargos de Pedagogia (Pedagogy)",
                options=roles,
                format_func=lambda r: r["name"],
                key="sync_pedagogy_roles_select"
            )

    st.markdown("</div>", unsafe_allow_html=True)

    if selected_role:
        role_id = selected_role["id"]

        if st.session_state.prev_role_id != role_id:
            st.session_state.prev_role_id = role_id
            st.session_state.fetched_members = None
            st.session_state.sync_metrics = None

        st.markdown("---")
        btn_col1, btn_col2, btn_col3 = st.columns([1, 2, 1])
        with btn_col2:
            trigger_search = st.button("🔄 Sincronizar & Buscar Alunos", use_container_width=True, key="sync_search_btn")

        if trigger_search:
            with st.spinner("Buscando e filtrando membros da API do Discord..."):
                members = fetch_members(guild_id=guild_id, role_id=role_id)
            
            if members is None:
                st.error("❌ Falha crítica ao buscar membros no Discord. Verifique os privilégios do Bot no servidor.")
                st.session_state.fetched_members = None
            elif not members:
                st.info(f"ℹ️ Nenhum aluno encontrado com o cargo **{selected_role['name']}** no servidor.")
                st.session_state.fetched_members = None
            else:
                st.session_state.fetched_members = members
                st.session_state.sync_metrics = None

        # Render search result table if cached
        if st.session_state.fetched_members is not None:
            members = st.session_state.fetched_members
            st.success("🎉 Membros do Cargo Primário sincronizados com sucesso!")

            met_col1, met_col2 = st.columns([1, 3])
            with met_col1:
                st.metric(
                    label="Alunos Ativos no Cargo",
                    value=len(members),
                    delta="Sincronizado"
                )

            df = pd.DataFrame(members)
            df = df.rename(columns={
                "id": "ID no Discord",
                "username": "Nome de Usuário",
                "nickname": "Apelido no Servidor"
            })
            df["Apelido no Servidor"] = df["Apelido no Servidor"].apply(
                lambda x: x if x and x.strip() else "(Sem apelido local)"
            )

            st.dataframe(df, use_container_width=True, hide_index=True)
            st.divider()

            st.markdown("<div class='section-title'>💾 Persistência Estendida no PocketBase</div>", unsafe_allow_html=True)
            
            if st.button("💾 Salvar e Sincronizar Tudo (PocketBase)", type="primary", use_container_width=True, key="sync_save_pb_btn"):
                managers_payload = []
                for r in selected_admin_roles:
                    managers_payload.append({"role_id": r["id"], "manager_type": "admin"})
                for r in selected_mentor_roles:
                    managers_payload.append({"role_id": r["id"], "manager_type": "mentor"})
                for r in selected_pedagogy_roles:
                    managers_payload.append({"role_id": r["id"], "manager_type": "pedagogy"})

                payload = {
                    "students": {
                        "primary_role_id": role_id,
                        "secondary_role_ids": [r["id"] for r in selected_secondary_roles]
                    },
                    "managers": managers_payload
                }

                with st.spinner("Persistindo dados no PocketBase (Advanced Upsert)..."):
                    result_data = sync_advanced_to_db(guild_id=guild_id, payload=payload)

                if result_data:
                    st.session_state.sync_metrics = result_data.get("metrics")
                    st.success("🎉 Sincronização avançada concluída com sucesso!")
                    # Clear caching to ensure the rest of the application loads fresh database state
                    st.cache_data.clear()
                    time.sleep(0.5)
                    st.rerun()
                else:
                    st.error("❌ **Falha na Persistência:** Não foi possível sincronizar com o PocketBase.")
                    st.session_state.sync_metrics = None

            # Render summary of mutations
            if st.session_state.sync_metrics is not None:
                metrics = st.session_state.sync_metrics
                st.markdown("<div class='card-section' style='margin-top: 16px;'>", unsafe_allow_html=True)
                st.markdown("<div class='section-title'>📊 Resumo das Mutações Avançadas</div>", unsafe_allow_html=True)
                
                st.markdown("<h5 style='color: #6366F1;'>🎓 Alunos / Students</h5>", unsafe_allow_html=True)
                col1, col2, col3 = st.columns(3)
                with col1:
                    st.metric("Total Processado", metrics.get("students_processed", 0))
                with col2:
                    st.metric("Novos Alunos", metrics.get("students_inserted", 0))
                with col3:
                    st.metric("Atualizados", metrics.get("students_updated", 0))
                    
                st.markdown("<h5 style='color: #EC4899; margin-top: 12px;'>👑 Equipe / Managers</h5>", unsafe_allow_html=True)
                col4, col5, col6 = st.columns(3)
                with col4:
                    st.metric("Total Processado", metrics.get("managers_processed", 0))
                with col5:
                    st.metric("Novos Managers", metrics.get("managers_inserted", 0))
                with col6:
                    st.metric("Atualizados", metrics.get("managers_updated", 0))
                    
                st.markdown("</div>", unsafe_allow_html=True)

# ==========================================
# ABA 2: Infraestrutura
# ==========================================
with tab2:
    st.markdown("### 🏗️ Provisionamento de Canais")
    st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top:-10px; margin-bottom: 20px;'>"
                "Crie categorias e canais 1-on-1 privados para os alunos do cargo monitorado selecionado.</p>",
                unsafe_allow_html=True)

    # FAIL-SAFE 1: Check if server has students synced
    if total_students_synced == 0:
        st.warning("⚠️ **Sincronize o servidor primeiro.** Execute a sincronização de alunos na Aba 1 para liberar esta aba.")
    else:
        # Extract variables from provision aggregate data
        p_roles = provision_data.get("roles", []) if provision_data else []
        p_categories = provision_data.get("categories", []) if provision_data else []
        p_text_channels = provision_data.get("text_channels", []) if provision_data else []
        announcement_channel_id = provision_data.get("announcement_channel_id", "") if provision_data else ""
        total_students_without_channels = provision_data.get("metrics", {}).get("total_students_without_channels", 0) if provision_data else 0

        # Mapeamento do Canal de Avisos
        st.markdown("<div class='card-section'>", unsafe_allow_html=True)
        st.markdown("<div class='section-title'>📢 Canal Oficial de Comunicados</div>", unsafe_allow_html=True)
        
        if not p_text_channels:
            st.warning("⚠️ Nenhum canal de texto encontrado neste servidor.")
        else:
            default_chan_idx = 0
            for idx, ch in enumerate(p_text_channels):
                if ch["id"] == announcement_channel_id:
                    default_chan_idx = idx
                    break
            
            col_chan_sel, col_chan_btn = st.columns([3, 1])
            with col_chan_sel:
                selected_announcement_chan = st.selectbox(
                    "Selecione o Canal de Avisos (Megafone)",
                    options=p_text_channels,
                    index=default_chan_idx,
                    format_func=lambda c: f"#{c['name']} (ID: {c['id']})",
                    key="setup_announcement_channel_select"
                )
            with col_chan_btn:
                st.markdown("<div style='margin-top: 28px;'></div>", unsafe_allow_html=True)
                if st.button("💾 Salvar Canal de Avisos", use_container_width=True, key="setup_save_announcement_btn"):
                    with st.spinner("Salva canal..."):
                        success, result = save_announcement_channel(guild_id, selected_announcement_chan["id"])
                        if success:
                            st.success("Canal de avisos atualizado!")
                            st.cache_data.clear()
                            time.sleep(0.5)
                            st.rerun()
                        else:
                            st.error(f"Erro: {result}")
        st.markdown("</div>", unsafe_allow_html=True)

        # Mapeamento de Turma para provisionamento
        st.markdown("<div class='card-section'>", unsafe_allow_html=True)
        st.markdown("<div class='section-title'>🏗️ Provisionar Canais por Cargo</div>", unsafe_allow_html=True)

        col_prov1, col_prov2 = st.columns(2)
        with col_prov1:
            if not p_roles:
                st.warning("⚠️ Nenhum cargo encontrado.")
                selected_prov_role = None
            else:
                selected_prov_role = st.selectbox(
                    "Selecione o Cargo (Turma):", 
                    options=p_roles, 
                    format_func=lambda r: r["name"],
                    key="setup_provision_role_select"
                )

        if total_students_without_channels > 0:
            st.info(f"💡 **Informação:** Existem **{total_students_without_channels}** alunos pendentes de canal de texto 1-on-1 privado.")

        if selected_prov_role:
            st.markdown("<div style='margin-top: 15px;'></div>", unsafe_allow_html=True)
            st.markdown("##### 📁 Estratégia de Categoria Pai")
            
            cat_strategy = st.radio(
                "Escolha a organização dos canais:",
                options=["📁 Usar Categoria Existente", "✨ Criar Nova Categoria"],
                horizontal=True,
                key="setup_cat_strategy_radio"
            )
            
            category_id = None
            
            if cat_strategy == "📁 Usar Categoria Existente":
                if not p_categories:
                    st.info("ℹ️ Nenhuma categoria encontrada no Discord. Escolha 'Criar Nova Categoria'.")
                else:
                    default_cat_idx = 0
                    if st.session_state.auto_select_category_id:
                        for idx, cat in enumerate(p_categories):
                            if cat["id"] == st.session_state.auto_select_category_id:
                                default_cat_idx = idx
                                break
                    
                    selected_cat = st.selectbox(
                        "Selecione a Categoria Pai",
                        options=p_categories,
                        format_func=lambda c: f"{c['name']} (ID: {c['id']})",
                        index=default_cat_idx,
                        key="setup_parent_cat_selectbox"
                    )
                    if selected_cat:
                        category_id = selected_cat["id"]
            else:
                col_cat1, col_cat2 = st.columns([3, 1])
                with col_cat1:
                    new_cat_name = st.text_input("Nome da Categoria", placeholder="Ex: 📁 CANAIS ALUNOS", key="setup_new_cat_name_input")
                with col_cat2:
                    new_cat_pos = st.number_input("Posição (0 = Topo)", min_value=0, value=0, key="setup_new_cat_pos_input")
                
                if st.button("✨ Criar Categoria no Discord", key="setup_create_cat_btn"):
                    if not new_cat_name.strip():
                        st.error("Nome da categoria não pode estar em branco!")
                    else:
                        with st.spinner("Criando categoria no Discord..."):
                            success, data = create_category(guild_id, new_cat_name.strip(), int(new_cat_pos))
                        if success:
                            st.session_state.auto_select_category_id = data.get("id")
                            st.success(f"Categoria '{new_cat_name}' criada!")
                            st.cache_data.clear()
                            time.sleep(0.5)
                            st.rerun()
                        else:
                            st.error(f"Erro: {data}")

            st.divider()

            # FAIL-SAFE 3: Disable provisioning if no monitored roles exist
            disable_prov_btn = not monitored_roles or (category_id is None)

            if disable_prov_btn:
                if not monitored_roles:
                    st.warning("⚠️ **Triagem Obrigatória:** Selecione ao menos uma turma monitorada na Aba 3 antes de provisionar.")
                elif category_id is None:
                    st.warning("⚠️ Selecione ou crie uma categoria pai para habilitar o botão.")

            if st.button("🚀 Provisionar Canais 1-on-1", type="primary", use_container_width=True, disabled=disable_prov_btn, key="setup_provision_run_btn"):
                st.session_state.provision_metrics = None
                with st.status("🏗️ Construindo infraestrutura de canais...", expanded=True) as status:
                    st.write("🔗 Conectando à API do Discord...")
                    st.write("⏳ Processando fila de alunos (cooldown de 800ms anti-spam)...")
                    
                    success, data = provision_channels(guild_id, category_id, selected_prov_role["id"])
                    
                    if success:
                        st.session_state.provision_metrics = data.get("metrics")
                        status.update(label="🎉 Provisionamento Concluído!", state="complete", expanded=False)
                        st.cache_data.clear()
                    else:
                        status.update(label="❌ Erro no Processamento!", state="error", expanded=True)
                        st.error(data)

            if st.session_state.provision_metrics:
                p_metrics = st.session_state.provision_metrics
                st.markdown("<div class='card-section' style='margin-top: 16px;'>", unsafe_allow_html=True)
                st.markdown("<div class='section-title'>📊 Métricas Finais do Lote</div>", unsafe_allow_html=True)
                
                met1, met2, met3, met4 = st.columns(4)
                with met1:
                    st.metric("Total de Alunos", p_metrics.get("total_students", 0))
                with met2:
                    st.metric("Canais Criados", p_metrics.get("channels_created", 0), delta="Criados")
                with met3:
                    st.metric("Já Provisionados", p_metrics.get("already_provisioned", 0), delta="Ignorados", delta_color="off")
                with met4:
                    st.metric("Erros Encontrados", p_metrics.get("errors", 0), delta="Pendentes" if p_metrics.get("errors", 0) > 0 else "Limpo", delta_color="inverse")
                st.markdown("</div>", unsafe_allow_html=True)

        st.markdown("</div>", unsafe_allow_html=True)

        # Disaster Recovery Auto-Healing
        st.markdown("<div style='margin-top: 32px;'></div>", unsafe_allow_html=True)
        with st.expander("🛠️ Zona de Recuperação de Desastres (Auto-Healing)"):
            st.warning("⚠️ **Recuperação de Desastres**\n\n"
                       "Faz varredura nos canais físicos sob a categoria selecionada e reassocia ao banco de dados pelo username.")
            
            if not p_categories:
                st.info("ℹ️ Nenhuma categoria encontrada no servidor Discord.")
            else:
                col_heal1, col_heal2 = st.columns([3, 1])
                with col_heal1:
                    heal_category = st.selectbox(
                        "Selecione a Categoria Pai para Varredura",
                        options=p_categories,
                        format_func=lambda c: f"{c['name']} (ID: {c['id']})",
                        key="setup_heal_category_select"
                    )
                with col_heal2:
                    st.markdown("<div style='margin-top: 28px;'></div>", unsafe_allow_html=True)
                    heal_btn = st.button("🔗 Reassociar Canais", type="secondary", use_container_width=True, key="setup_heal_run_btn")
                
                if heal_btn and heal_category:
                    st.session_state.heal_metrics = None
                    with st.spinner("Curando a base de dados..."):
                        success, data = heal_channels(guild_id, heal_category["id"])
                        if success:
                            st.session_state.heal_metrics = data.get("metrics")
                            st.success("🎉 Auto-healing concluído!")
                            st.cache_data.clear()
                        else:
                            st.error(f"❌ Erro na recuperação: {data}")
            
            if st.session_state.heal_metrics:
                h_metrics = st.session_state.heal_metrics
                st.markdown("<div class='card-section' style='margin-top: 16px;'>", unsafe_allow_html=True)
                st.markdown("<div class='section-title'>📊 Resultados do Auto-Healing</div>", unsafe_allow_html=True)
                
                hcol1, hcol2, hcol3, hcol4 = st.columns(4)
                with hcol1:
                    st.metric("Canais Escaneados", h_metrics.get("channels_scanned", 0))
                with hcol2:
                    st.metric("Mapeados", h_metrics.get("successfully_mapped", 0), delta="Curados")
                with hcol3:
                    st.metric("Canais Sem Aluno", h_metrics.get("unmapped_channels", 0), delta="Ignorados", delta_color="off")
                with hcol4:
                    st.metric("Alunos Sem Canal", h_metrics.get("students_still_pending", 0), delta="Pendentes" if h_metrics.get("students_still_pending", 0) > 0 else "Completo", delta_color="inverse")
                st.markdown("</div>", unsafe_allow_html=True)

# ==========================================
# ABA 3: Regras de Ponto
# ==========================================
with tab3:
    st.markdown("### ⏰ Regras de Ponto e Triagem")
    st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top:-10px; margin-bottom: 20px;'>"
                "Defina quais cargos representam turmas monitoradas, configure seus turnos e teste o bot no Sandbox.</p>",
                unsafe_allow_html=True)

    # FAIL-SAFE 1: Check if server has students synced
    if total_students_synced == 0:
        st.warning("⚠️ **Sincronize o servidor primeiro.** Execute a sincronização de alunos na Aba 1 para liberar esta aba.")
    else:
        # Timezone clock header
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

        # Tab sections
        subtab1, subtab2 = st.tabs(["⏰ Horários de Ponto", "⚙️ Triagem de Turmas"])

        # SUBTAB 1: Horários de Ponto
        with subtab1:
            if not monitored_roles:
                st.info("ℹ️ Nenhuma turma foi selecionada para monitoramento ainda. Acesse a aba '⚙️ Triagem de Turmas' ao lado para habilitar.")
            else:
                st.markdown('<h4 style="font-size: 1.2rem; font-weight: 600; color: #F1F5F9; margin-bottom: 16px;">📚 Horários das Turmas Monitoradas</h4>', unsafe_allow_html=True)
                
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

                    # Card layout
                    st.markdown(f'<div class="card-section">', unsafe_allow_html=True)
                    
                    tcol1, tcol2 = st.columns([3, 1])
                    with tcol1:
                        st.markdown(f'<div class="section-title" style="margin-bottom:0px;">🎓 Turma: <b>{role_discord_name}</b> <span style="font-size:0.85rem; font-weight:normal; color:#64748B;">({role_pb_id})</span></div>', unsafe_allow_html=True)
                    with tcol2:
                        val_active = st.toggle(
                            "Agendamento Ativo",
                            value=current_is_active,
                            key=f"active_setup_{role_pb_id}_{index}"
                        )

                    st.markdown('<div style="height:15px;"></div>', unsafe_allow_html=True)
                    
                    col1, col2, col3 = st.columns(3)
                    with col1:
                        sel_shift_lbl = st.selectbox(
                            "Turno de Estudo",
                            options=["Manhã", "Tarde", "Noite"],
                            index=["Manhã", "Tarde", "Noite"].index(current_shift_lbl),
                            key=f"shift_setup_{role_pb_id}_{index}"
                        )
                    with col2:
                        val_check_in = st.text_input(
                            "Horário de Entrada (Disparo)",
                            value=current_check_in,
                            placeholder="HH:MM",
                            key=f"checkin_setup_{role_pb_id}_{index}"
                        )
                    with col3:
                        val_cooldown = st.number_input(
                            "Janela de Saída em Horas",
                            min_value=1,
                            max_value=12,
                            value=current_cooldown,
                            key=f"cooldown_setup_{role_pb_id}_{index}"
                        )

                    st.write("")
                    col_btn, _ = st.columns([1, 4])
                    with col_btn:
                        save_btn = st.button(
                            f"💾 Salvar Configurações",
                            key=f"save_setup_{role_pb_id}_{index}",
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
                                st.cache_data.clear()
                                time.sleep(0.5)
                                st.rerun()
                            else:
                                st.error(f"❌ Erro ao salvar configurações: {result}")
                    
                    st.markdown('</div>', unsafe_allow_html=True)

        # SUBTAB 2: Triagem de Turmas
        with subtab2:
            st.markdown('<h4>⚙️ Triagem de Cargos do Discord</h4>', unsafe_allow_html=True)
            st.markdown('<p style="color: #94A3B8; font-size: 0.95rem;">Defina quais cargos representam turmas de alunos que participarão do controle de ponto diário.</p>', unsafe_allow_html=True)

            st.markdown('<div class="card-section">', unsafe_allow_html=True)
            
            # Fetch all roles configurations
            if not roles_config:
                st.warning("Nenhum cargo sincronizado no banco de dados.")
            else:
                current_monitored_ids = [r["id"] for r in roles_config if r.get("is_monitored", False)]
                
                selected_role_ids = st.multiselect(
                    "Selecione os cargos/turmas monitorados:",
                    options=[r["id"] for r in roles_config],
                    default=current_monitored_ids,
                    format_func=lambda rid: next((r["name"] for r in roles_config if r["id"] == rid), rid),
                    key="setup_triagem_multiselect"
                )

                st.write("")
                col_save_triagem, _ = st.columns([1, 3])
                with col_save_triagem:
                    save_triagem_btn = st.button("💾 Salvar Triagem de Turmas", use_container_width=True, key="setup_save_triagem_btn")

                if save_triagem_btn:
                    success_count = 0
                    errors = []
                    
                    with st.spinner("Atualizando triagem de turmas..."):
                        for r in roles_config:
                            rid = r["id"]
                            should_be_monitored = rid in selected_role_ids
                            was_monitored = r.get("is_monitored", False)
                            
                            if should_be_monitored != was_monitored:
                                payload = {"is_monitored": should_be_monitored}
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
                        st.success(f"✅ Triagem de turmas atualizada!")
                        st.cache_data.clear()
                        time.sleep(0.5)
                        st.rerun()

            st.markdown('</div>', unsafe_allow_html=True)

        # Sandbox Dry Run
        st.markdown('<br>', unsafe_allow_html=True)
        with st.expander("🧪 Sandbox de Testes (Dry Run)", expanded=False):
            st.markdown(
                '<p style="color: #94A3B8; font-size: 0.95rem; margin-bottom: 16px;">'
                'Simule o envio do botão oficial de Check-In/Check-Out em um canal de testes do Discord.'
                '</p>',
                unsafe_allow_html=True
            )

            managers = fetch_guild_managers(guild_id)
            if not managers:
                st.warning("⚠️ Nenhum manager/mentor cadastrado para esta guilda. Sincronize os usuários primeiro na Aba 1.")
            else:
                manager_options = {m["discord_id"]: f"{m['name']} ({m.get('role', 'mentor')})" for m in managers}
                
                mcol1, mcol2 = st.columns(2)
                with mcol1:
                    selected_tester_id = st.selectbox(
                        "Quem vai testar o botão? (Equipe/Manager)",
                        options=list(manager_options.keys()),
                        format_func=lambda mid: manager_options[mid],
                        key="setup_tester_selectbox"
                    )
                with mcol2:
                    test_channel_id = st.text_input(
                        "ID do Canal de Teste no Discord",
                        placeholder="Ex: 112233445566778899",
                        key="setup_test_channel_id_input"
                    )

                st.write("")
                col_trigger, _ = st.columns([1, 3])
                with col_trigger:
                    trigger_btn = st.button("🚀 Disparar Teste de Ponto", use_container_width=True, type="secondary", key="setup_trigger_test_btn")

                if trigger_btn:
                    if not test_channel_id.strip():
                        st.error("❌ Por favor, informe um ID de Canal do Discord válido.")
                    else:
                        with st.spinner("Disparando mensagem de ponto para o canal..."):
                            success, msg = trigger_test_attendance(
                                guild_id,
                                test_channel_id.strip(),
                                selected_tester_id
                            )
                            if success:
                                st.success(f"✅ Dry run acionado! {msg}")
                            else:
                                st.error(f"❌ Erro ao disparar teste: {msg}")

# ==========================================
# ABA 4: Estrutura e Squads
# ==========================================
with tab4:
    st.markdown("### 👥 Gestão de Estrutura de Squads")
    st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top:-10px; margin-bottom: 20px;'>"
                "Vincule canais específicos do Discord para cada Squad, consulte métricas de engajamento e a listagem de alunos.</p>",
                unsafe_allow_html=True)

    # FAIL-SAFE 1: Check if server has students synced
    if total_students_synced == 0:
        st.warning("⚠️ **Sincronize o servidor primeiro.** Execute a sincronização de alunos na Aba 1 para liberar esta aba.")
    # FAIL-SAFE 2: Check if monitored roles are configured
    elif not monitored_roles:
        st.info("ℹ️ **Nenhuma turma monitorada neste servidor.** Por favor, selecione e salve os cargos monitorados na Aba 3 (Regras de Ponto) primeiro.")
    else:
        # Monitored role selector
        role_opts = {r['id']: f"{r['name']} (Turno: {r.get('shift', 'N/A')})" for r in monitored_roles}
        selected_role_id = st.selectbox(
            "Selecione a Turma (Squad):", 
            options=list(role_opts.keys()), 
            format_func=lambda x: role_opts[x],
            key="squad_role_selector"
        )

        st.markdown("---")

        if selected_role_id:
            with st.spinner("Carregando informações da turma..."):
                dashboard_data = fetch_squad_dashboard_data(selected_role_id)
                
            if not dashboard_data:
                st.error("Erro ao carregar dados da turma. Verifique a conexão com o servidor.")
            else:
                squad_info = dashboard_data.get("squad_info", {})
                s_metrics = dashboard_data.get("metrics", {})
                text_channels = dashboard_data.get("text_channels", [])
                students = dashboard_data.get("students", [])

                # 1. Official Channel Config
                st.markdown("#### ⚙️ Canal Oficial da Turma")
                with st.container(border=True):
                    col_ch, col_btn = st.columns([0.7, 0.3])
                    
                    channel_opts = {"": "Nenhum canal vinculado"}
                    for ch in text_channels:
                        channel_opts[ch['id']] = f"#{ch['name']}"
                        
                    current_channel = squad_info.get("squad_channel_id", "")
                    
                    with col_ch:
                        new_channel_id = st.selectbox(
                            "Selecione o canal de texto base desta turma no Discord:",
                            options=list(channel_opts.keys()),
                            format_func=lambda x: channel_opts[x],
                            index=list(channel_opts.keys()).index(current_channel) if current_channel in channel_opts else 0,
                            key="squad_channel_link_select"
                        )
                        
                    with col_btn:
                        st.write("")
                        st.write("")
                        if st.button("💾 Vincular Canal", use_container_width=True, type="primary", key="link_squad_chan_btn"):
                            if new_channel_id != current_channel:
                                with st.spinner("Salvando..."):
                                    success, msg = update_squad_channel(selected_role_id, new_channel_id)
                                    if success:
                                        st.success("Canal vinculado com sucesso!")
                                        st.cache_data.clear()
                                        time.sleep(0.5)
                                        st.rerun()
                                    else:
                                        st.error(f"Erro ao salvar: {msg}")
                            else:
                                st.info("O canal selecionado já é o atual.")

                st.markdown("<br>", unsafe_allow_html=True)

                # 2. Metrics overview
                st.markdown("#### 📊 Visão Geral")
                m1, m2, m3 = st.columns(3)
                
                m1.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value">{s_metrics.get("total_students", 0)}</div>
                    <div class="metric-label">Total de Alunos</div>
                </div>
                """, unsafe_allow_html=True)
                
                m2.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: #38BDF8;">{s_metrics.get("provisioned_1on1_channels", 0)}</div>
                    <div class="metric-label">Canais 1-on-1 Criados</div>
                </div>
                """, unsafe_allow_html=True)
                
                rate = s_metrics.get("average_attendance_percent", 0.0)
                rate_color = "#34D399" if rate >= 75.0 else ("#FBBF24" if rate >= 50 else "#F87171")
                m3.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: {rate_color};">{rate:.1f}%</div>
                    <div class="metric-label">Média de Presença (30d)</div>
                </div>
                """, unsafe_allow_html=True)

                st.markdown("<br>", unsafe_allow_html=True)

                # 3. Students list
                st.markdown("#### 🧑‍🎓 Lista de Alunos")
                if not students:
                    st.info("Nenhum aluno ativo encontrado nesta turma.")
                else:
                    df = pd.DataFrame(students)
                    df["has_1on1_ui"] = df["has_1on1"].apply(lambda x: "🟢 Pronto" if x else "🔴 Pendente")
                    
                    df_display = df[["username", "nickname", "has_1on1_ui"]].copy()
                    df_display.rename(columns={
                        "username": "Discord Username",
                        "nickname": "Nome / Apelido",
                        "has_1on1_ui": "Status Canal 1-on-1"
                    }, inplace=True)
                    
                    st.dataframe(df_display, use_container_width=True, hide_index=True)
