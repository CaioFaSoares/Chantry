import streamlit as st
import requests
import socket
import time

# 1. Page Config
st.set_page_config(
    page_title="Provisionamento de Infraestrutura - Chantry",
    page_icon="🏗️",
    layout="wide",
    initial_sidebar_state="expanded"
)

# 2. Premium CSS (Outfit font, gradients, and cards)
custom_css = """
<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap');

html, body, [class*="css"], .stMarkdown, .stButton button {
    font-family: 'Outfit', sans-serif !important;
}

.main-header {
    background: linear-gradient(135deg, #10B981 0%, #3B82F6 50%, #8B5CF6 100%);
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
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

from utils.api_client import (
    fetch_guilds,
    fetch_roles,
    fetch_categories,
    create_category,
    provision_channels,
    heal_channels,
    get_provision_page_data,
    save_announcement_channel
)

# 4. Session State Setup
if "auto_select_category_id" not in st.session_state:
    st.session_state.auto_select_category_id = None
if "provision_metrics" not in st.session_state:
    st.session_state.provision_metrics = None
if "heal_metrics" not in st.session_state:
    st.session_state.heal_metrics = None
if "selected_guild_id" not in st.session_state:
    st.session_state.selected_guild_id = None

# Sidebar
st.sidebar.image("https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=300&q=80", width=True)
st.sidebar.markdown("<h2 style='text-align: center;'>Chantry Suite</h2>", unsafe_allow_html=True)
st.sidebar.markdown("---")

# Title
st.markdown("<h1 class='main-header'>🏗️ Provisionamento de Infraestrutura</h1>", unsafe_allow_html=True)
st.markdown("<p class='subtitle'>Geração automática de categorias e canais 1-on-1 privados para turmas integradas</p>", unsafe_allow_html=True)

# 5. Core Interface
guilds = fetch_guilds()
if guilds is None:
    st.error("❌ **Erro de Conexão:** Não foi possível conectar ao Go Server Daemon na porta 12000.")
elif not guilds:
    st.warning("⚠️ **Sem Servidores:** O bot não está presente em nenhuma guilda autorizada.")
else:
    st.markdown("<div class='card-section'>", unsafe_allow_html=True)
    st.markdown("<div class='section-title'>⚙️ 1. Mapeamento de Servidor & Turma</div>", unsafe_allow_html=True)
    
    # Preserve selected guild in session state
    default_index = 0
    if st.session_state.selected_guild_id and any(g["id"] == st.session_state.selected_guild_id for g in guilds):
        default_index = next(i for i, g in enumerate(guilds) if g["id"] == st.session_state.selected_guild_id)

    col1, col2 = st.columns(2)
    with col1:
        selected_guild = st.selectbox(
            "Selecione o Servidor (Guild)",
            options=guilds,
            index=default_index,
            format_func=lambda g: g["name"]
        )
    
    if selected_guild:
        guild_id = selected_guild["id"]
        st.session_state.selected_guild_id = guild_id
        
        # Load BFF aggregate page data
        page_data = get_provision_page_data(guild_id)
        if page_data is None:
            st.error("❌ **Erro:** Não foi possível carregar os dados agregados da guilda do BFF.")
            roles = []
            categories = []
            text_channels = []
            announcement_channel_id = ""
            total_students_without_channels = 0
        else:
            roles = page_data.get("roles", [])
            categories = page_data.get("categories", [])
            text_channels = page_data.get("text_channels", [])
            announcement_channel_id = page_data.get("announcement_channel_id", "")
            metrics = page_data.get("metrics", {})
            total_students_without_channels = metrics.get("total_students_without_channels", 0)

        with col2:
            if not roles:
                st.warning("⚠️ Nenhum cargo encontrado.")
                selected_role = None
            else:
                selected_role = st.selectbox("Selecione o Cargo (Turma)", options=roles, format_func=lambda r: r["name"])
        st.markdown("</div>", unsafe_allow_html=True)

        # Show informational alert if students are pending private channel setup
        if page_data and total_students_without_channels > 0:
            st.info(f"💡 **Informação:** Existem **{total_students_without_channels}** alunos pendentes de provisionamento de canal nesta guilda.")

        # 1.1 Announcement Channel configuration (Épico 6)
        if page_data:
            st.markdown("<div class='card-section'>", unsafe_allow_html=True)
            st.markdown("<div class='section-title'>📢 Canal Oficial de Comunicados</div>", unsafe_allow_html=True)
            st.markdown("<p style='color: #94A3B8; font-size: 0.95rem; margin-top: -10px; margin-bottom: 15px;'>"
                        "Selecione qual canal de texto funcionará como o megafone oficial para comunicados gerais da guilda.</p>",
                        unsafe_allow_html=True)
            
            if not text_channels:
                st.warning("⚠️ Nenhum canal de texto encontrado neste servidor.")
            else:
                # Find current index for pre-selection
                default_chan_idx = 0
                for idx, ch in enumerate(text_channels):
                    if ch["id"] == announcement_channel_id:
                        default_chan_idx = idx
                        break
                
                col_chan_sel, col_chan_btn = st.columns([3, 1])
                with col_chan_sel:
                    selected_announcement_chan = st.selectbox(
                        "Selecione o Canal de Avisos",
                        options=text_channels,
                        index=default_chan_idx,
                        format_func=lambda c: f"#{c['name']} (ID: {c['id']})",
                        key="announcement_channel_selectbox"
                    )
                with col_chan_btn:
                    st.markdown("<div style='margin-top: 28px;'></div>", unsafe_allow_html=True)
                    if st.button("💾 Salvar Canal de Avisos", use_container_width=True):
                        with st.spinner("Persistindo configuração..."):
                            success, result = save_announcement_channel(guild_id, selected_announcement_chan["id"])
                            if success:
                                st.success("🎉 Canal de avisos atualizado com sucesso!")
                                time.sleep(1)
                                st.rerun()
                            else:
                                st.error(f"Erro ao salvar: {result}")
            st.markdown("</div>", unsafe_allow_html=True)

        if selected_role:
            role_id = selected_role["id"]
            
            # 6. Category Strategy Layout
            st.markdown("<div class='card-section'>", unsafe_allow_html=True)
            st.markdown("<div class='section-title'>📁 2. Estratégia de Categoria Pai</div>", unsafe_allow_html=True)
            
            cat_strategy = st.radio(
                "Escolha como organizar os canais de texto:",
                options=["📁 Usar Categoria Existente", "✨ Criar Nova Categoria"],
                horizontal=True
            )
            
            category_id = None
            
            if cat_strategy == "📁 Usar Categoria Existente":
                if not categories:
                    st.info("ℹ️ Nenhuma categoria encontrada no servidor Discord. Por favor, selecione 'Criar Nova Categoria' abaixo.")
                else:
                    default_idx = 0
                    if st.session_state.auto_select_category_id:
                        for idx, cat in enumerate(categories):
                            if cat["id"] == st.session_state.auto_select_category_id:
                                default_idx = idx
                                break
                    
                    selected_cat = st.selectbox(
                        "Selecione a Categoria Pai",
                        options=categories,
                        format_func=lambda c: f"{c['name']} (ID: {c['id']})",
                        index=default_idx
                    )
                    if selected_cat:
                        category_id = selected_cat["id"]
            
            else:
                col_cat1, col_cat2 = st.columns([3, 1])
                with col_cat1:
                    new_cat_name = st.text_input("Nome da Nova Categoria", placeholder="Ex: 📁 CANAIS ALUNOS")
                with col_cat2:
                    new_cat_pos = st.number_input("Posição na Lista (0 = Topo)", min_value=0, value=0, step=1)
                
                if st.button("✨ Criar Categoria no Discord"):
                    if not new_cat_name.strip():
                        st.error("Por favor, preencha o nome da categoria!")
                    else:
                        with st.spinner("Criando categoria no Discord..."):
                            success, data = create_category(guild_id, new_cat_name.strip(), int(new_cat_pos))
                        if success:
                            st.session_state.auto_select_category_id = data.get("id")
                            st.success(f"Categoria '{new_cat_name}' criada com sucesso!")
                            time.sleep(1)
                            st.rerun()
                        else:
                            st.error(f"Erro ao criar categoria: {data}")
            
            st.markdown("</div>", unsafe_allow_html=True)
            
            # 7. Action Block & Progress Logger
            st.divider()
            
            if category_id:
                if st.button("🚀 Provisionar Canais 1-on-1", type="primary", use_container_width=True):
                    st.session_state.provision_metrics = None
                    
                    # Status logger expander
                    with st.status("🏗️ Construindo infraestrutura de canais...", expanded=True) as status:
                        st.write("🔗 Conectando à API do Discord e resolvendo tabelas no banco de dados...")
                        st.write("⏳ O processamento está rodando em lote. Aplicando cooldowns de segurança (800ms) para evitar rate limits...")
                        
                        success, data = provision_channels(guild_id, category_id, role_id)
                        
                        if success:
                            st.session_state.provision_metrics = data.get("metrics")
                            status.update(label="🎉 Provisionamento Concluído!", state="complete", expanded=False)
                        else:
                            status.update(label="❌ Erro no Processamento!", state="error", expanded=True)
                            st.error(data)
                
                # Metrics layout
                if st.session_state.provision_metrics:
                    metrics = st.session_state.provision_metrics
                    st.markdown("<div class='card-section' style='margin-top: 16px;'>", unsafe_allow_html=True)
                    st.markdown("<div class='section-title'>📊 Métricas Finais do Lote</div>", unsafe_allow_html=True)
                    
                    met1, met2, met3, met4 = st.columns(4)
                    with met1:
                        st.metric("Total de Alunos", metrics.get("total_students", 0))
                    with met2:
                        st.metric("Canais Criados", metrics.get("channels_created", 0), delta="Criados", delta_color="normal")
                    with met3:
                        st.metric("Já Provisionados", metrics.get("already_provisioned", 0), delta="Ignorados", delta_color="off")
                    with met4:
                        st.metric("Erros Encontrados", metrics.get("errors", 0), delta="Pendentes" if metrics.get("errors", 0) > 0 else "Limpo", delta_color="inverse")
                    
                    st.markdown("</div>", unsafe_allow_html=True)

            # 8. Disaster Recovery Auto-Healing Expander
            st.markdown("<div style='margin-top: 32px;'></div>", unsafe_allow_html=True)
            with st.expander("🛠️ Zona de Recuperação de Desastres (Auto-Healing)"):
                st.warning("⚠️ **Recuperação de Desastres / Reassociação de Canais**\n\n"
                           "Use esta ferramenta caso a base de dados tenha sido limpa ou reiniciada, mas os canais "
                           "físicos no Discord ainda existam. O Chantry fará uma varredura na categoria selecionada, "
                           "mapeará os canais de volta aos alunos baseado no username e atualizará o banco de dados automaticamente.")
                
                if not categories:
                    st.info("ℹ️ Nenhuma categoria encontrada no servidor Discord.")
                else:
                    col_heal1, col_heal2 = st.columns([3, 1])
                    with col_heal1:
                        heal_category = st.selectbox(
                            "Selecione a Categoria com os Canais Órfãos",
                            options=categories,
                            format_func=lambda c: f"{c['name']} (ID: {c['id']})",
                            key="heal_category_select"
                        )
                    with col_heal2:
                        st.markdown("<div style='margin-top: 28px;'></div>", unsafe_allow_html=True)
                        heal_btn = st.button("🔗 Reassociar Canais Existentes", type="secondary", use_container_width=True)
                    
                    if heal_btn and heal_category:
                        st.session_state.heal_metrics = None
                        with st.spinner("Executando varredura e curando a base de dados..."):
                            success, data = heal_channels(guild_id, heal_category["id"])
                            if success:
                                st.session_state.heal_metrics = data.get("metrics")
                                st.success("🎉 Processo de auto-healing concluído com sucesso!")
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
                        st.metric("Mapeados com Sucesso", h_metrics.get("successfully_mapped", 0), delta="Curados", delta_color="normal")
                    with hcol3:
                        st.metric("Canais sem Aluno", h_metrics.get("unmapped_channels", 0), delta="Ignorados", delta_color="off")
                    with hcol4:
                        st.metric("Alunos ainda sem Canal", h_metrics.get("students_still_pending", 0), delta="Pendentes" if h_metrics.get("students_still_pending", 0) > 0 else "Completo", delta_color="inverse")
                    
                    st.markdown("</div>", unsafe_allow_html=True)
