import streamlit as st
import pandas as pd
from utils.api_client import fetch_guilds, fetch_roles, fetch_members, sync_advanced_to_db

# Page config
st.set_page_config(
    page_title="Sincronização Discord - Chantry",
    page_icon="👾",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Premium stylesheet inherited from Chantry app.py
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
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

# Sidebar image and navigation branding
st.sidebar.image("https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=300&q=80", width=True)
st.sidebar.markdown("<h2 style='text-align: center;'>Chantry Suite</h2>", unsafe_allow_html=True)
st.sidebar.markdown("---")

# Main Header Title
st.markdown("<h1 class='main-header'>👾 Sincronização Discord</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>Mapeamento reativo e avançado de alunos, skills e equipe baseados em cargos</p>", unsafe_allow_html=True)

# Initialize session states for page routing and persistence logic
if "fetched_members" not in st.session_state:
    st.session_state.fetched_members = None
if "sync_metrics" not in st.session_state:
    st.session_state.sync_metrics = None
if "prev_guild_id" not in st.session_state:
    st.session_state.prev_guild_id = None
if "prev_role_id" not in st.session_state:
    st.session_state.prev_role_id = None
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None

# Perform initial fetch from central client
guilds = fetch_guilds()

if guilds is None:
    st.error(
        "❌ **Erro de Conexão:** Não foi possível conectar ao Daemon do Go Server. "
        "Verifique se o serviço `go-server` está ativo na porta `12000` do Docker Compose."
    )
elif not guilds:
    st.warning(
        "⚠️ **Nenhum Servidor Identificado:** O bot do Chantry não foi adicionado a nenhum "
        "servidor ou o token não possui permissões adequadas de leitura."
    )
else:
    # Render form within a premium glassmorphic section container
    st.markdown("<div class='card-section'>", unsafe_allow_html=True)
    st.markdown("<div class='section-title'>⚙️ Configurações da Sincronização Avançada</div>", unsafe_allow_html=True)
    
    col_input1, col_input2 = st.columns(2)
    
    # Preserve selected guild in session state
    default_index = 0
    if st.session_state.selected_guild_id and any(g["id"] == st.session_state.selected_guild_id for g in guilds):
        default_index = next(i for i, g in enumerate(guilds) if g["id"] == st.session_state.selected_guild_id)

    with col_input1:
        selected_guild = st.selectbox(
            "1. Selecione o Servidor (Guild)",
            options=guilds,
            index=default_index,
            format_func=lambda g: g["name"],
            key="selected_guild_dropdown"
        )
    
    # Cascade reactivity: retrieve roles only if a server is selected
    if selected_guild:
        guild_id = selected_guild["id"]
        st.session_state.selected_guild_id = guild_id
        
        # Cascade state reset on guild switch
        if st.session_state.prev_guild_id != guild_id:
            st.session_state.prev_guild_id = guild_id
            st.session_state.fetched_members = None
            st.session_state.sync_metrics = None
            
        roles = fetch_roles(guild_id)
        
        with col_input2:
            if roles is None:
                st.error("❌ Erro ao buscar os cargos deste servidor.")
                selected_role = None
            elif not roles:
                st.warning("⚠️ Nenhum cargo personalizado encontrado.")
                selected_role = None
            else:
                selected_role = st.selectbox(
                    "2. Cargo Primário de Alunos (Squad/Turma)",
                    options=roles,
                    format_func=lambda r: r["name"],
                    key="selected_role_dropdown"
                )
        
        # Multiselect for secondary skill/path roles
        selected_secondary_roles = []
        if roles:
            selected_secondary_roles = st.multiselect(
                "3. Cargos Secundários de Alunos (Skills / Trilhas)",
                options=roles,
                format_func=lambda r: r["name"],
                key="selected_secondary_roles"
            )
            
            # Expander for Team/Manager mappings
            st.markdown("<div style='margin-top: 12px;'></div>", unsafe_allow_html=True)
            with st.expander("⚙️ Mapeamento de Equipe / Managers"):
                st.markdown("<p style='color: #94A3B8; font-size: 0.9rem;'>Associe cargos do Discord aos níveis de acesso de equipe no PocketBase:</p>", unsafe_allow_html=True)
                selected_admin_roles = st.multiselect(
                    "Cargos de Administradores (Admin)",
                    options=roles,
                    format_func=lambda r: r["name"],
                    key="admin_roles_multiselect"
                )
                selected_mentor_roles = st.multiselect(
                    "Cargos de Mentores (Mentor)",
                    options=roles,
                    format_func=lambda r: r["name"],
                    key="mentor_roles_multiselect"
                )
                selected_pedagogy_roles = st.multiselect(
                    "Cargos de Pedagogia (Pedagogy)",
                    options=roles,
                    format_func=lambda r: r["name"],
                    key="pedagogy_roles_multiselect"
                )
        
        st.markdown("</div>", unsafe_allow_html=True) # Closes card-section HTML block
        
        # Trigger member synchronization only if primary role is valid
        if selected_role:
            role_id = selected_role["id"]
            
            # Cascade state reset on primary role switch
            if st.session_state.prev_role_id != role_id:
                st.session_state.prev_role_id = role_id
                st.session_state.fetched_members = None
                st.session_state.sync_metrics = None
            
            st.markdown("---")
            
            # Center and expand trigger button
            btn_col1, btn_col2, btn_col3 = st.columns([1, 2, 1])
            with btn_col2:
                trigger_search = st.button(
                    "🔄 Sincronizar & Buscar Alunos",
                    use_container_width=True
                )
            
            if trigger_search:
                with st.spinner("Buscando e filtrando membros da API do Discord (processando paginação)..."):
                    members = fetch_members(guild_id=guild_id, role_id=role_id)
                
                if members is None:
                    st.error("❌ Falha crítica ao buscar membros no Discord. Verifique os privilégios do Bot no servidor.")
                    st.session_state.fetched_members = None
                elif not members:
                    st.info(f"ℹ️ Nenhum aluno encontrado com o cargo **{selected_role['name']}** no servidor.")
                    st.session_state.fetched_members = None
                else:
                    st.session_state.fetched_members = members
                    st.session_state.sync_metrics = None  # Reset metrics on new search
            
            # Persistent render logic based on session state
            if st.session_state.fetched_members is not None:
                members = st.session_state.fetched_members
                st.success("🎉 Membros do Cargo Primário sincronizados com sucesso!")
                
                # Modern metrics display
                met_col1, met_col2 = st.columns([1, 3])
                with met_col1:
                    st.metric(
                        label="Alunos Ativos no Cargo Primário",
                        value=len(members),
                        delta="Sincronizado"
                    )
                
                # Convert mapped array to high-performance Pandas DataFrame
                df = pd.DataFrame(members)
                
                # Format user column titles for professional dashboard UI
                df = df.rename(columns={
                    "id": "ID no Discord",
                    "username": "Nome de Usuário",
                    "nickname": "Apelido no Servidor"
                })
                
                # Empty cell formatting (e.g. empty string to placeholder)
                df["Apelido no Servidor"] = df["Apelido no Servidor"].apply(
                    lambda x: x if x and x.strip() else "(Sem apelido local)"
                )
                
                # Display structured data table with container resizing
                st.dataframe(
                    df,
                    use_container_width=True,
                    hide_index=True
                )
                
                st.divider()
                
                st.markdown("<div class='section-title'>💾 Persistência Estendida no PocketBase</div>", unsafe_allow_html=True)
                
                # Action button to trigger the persistence in database
                if st.button("💾 Salvar e Sincronizar Tudo (PocketBase)", type="primary", use_container_width=True):
                    # Construct advanced sync payload
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
                    else:
                        st.error("❌ **Falha na Persistência:** Não foi possível sincronizar com o PocketBase.")
                        st.session_state.sync_metrics = None
                
                # Render dual metrics dashboard if present
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
    else:
        st.markdown("</div>", unsafe_allow_html=True)
