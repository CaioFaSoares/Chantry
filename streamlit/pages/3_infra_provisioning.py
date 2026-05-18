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

# 3. API Handlers
def get_backend_url():
    try:
        socket.gethostbyname("go-server")
        return "http://go-server:12000/api"
    except socket.gaierror:
        return "http://localhost:12000/api"

base_url = get_backend_url()

def fetch_guilds():
    try:
        resp = requests.get(f"{base_url}/discord/guilds", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return None

def fetch_roles(guild_id):
    try:
        resp = requests.get(f"{base_url}/discord/guilds/{guild_id}/roles", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

def fetch_categories(guild_id):
    try:
        resp = requests.get(f"{base_url}/discord/guilds/{guild_id}/categories", timeout=5.0)
        return resp.json() if resp.status_code == 200 else []
    except Exception:
        return []

def create_category(guild_id, name, position):
    try:
        resp = requests.post(
            f"{base_url}/discord/guilds/{guild_id}/categories",
            json={"name": name, "position": position},
            timeout=10.0
        )
        return {"success": resp.status_code == 200, "data": resp.json() if resp.status_code == 200 else resp.text}
    except Exception as e:
        return {"success": False, "error": str(e)}

def provision_channels(guild_id, category_id, role_id):
    try:
        resp = requests.post(
            f"{base_url}/provision/guilds/{guild_id}/channels",
            json={"category_id": category_id, "role_id": role_id},
            timeout=180.0 # Long timeout for batch execution
        )
        if resp.status_code == 200:
            return {"success": True, "data": resp.json()}
        else:
            try:
                err_msg = resp.json().get("error", "Unknown error")
            except Exception:
                err_msg = resp.text
            return {"success": False, "error": err_msg}
    except requests.exceptions.Timeout:
        return {"success": False, "error": "Requisição expirou no Streamlit (Timeout). Verifique se o lote está executando no Go Daemon."}
    except Exception as e:
        return {"success": False, "error": str(e)}

# 4. Session State Setup
if "auto_select_category_id" not in st.session_state:
    st.session_state.auto_select_category_id = None
if "provision_metrics" not in st.session_state:
    st.session_state.provision_metrics = None

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
    st.error("❌ **Erro de Conexão:** Não foi possível conectar ao Go Backend Daemon na porta 12000.")
elif not guilds:
    st.warning("⚠️ **Sem Servidores:** O bot não está presente em nenhuma guilda autorizada.")
else:
    st.markdown("<div class='card-section'>", unsafe_allow_html=True)
    st.markdown("<div class='section-title'>⚙️ 1. Mapeamento de Servidor & Turma</div>", unsafe_allow_html=True)
    
    col1, col2 = st.columns(2)
    with col1:
        selected_guild = st.selectbox("Selecione o Servidor (Guild)", options=guilds, format_func=lambda g: g["name"])
    
    if selected_guild:
        guild_id = selected_guild["id"]
        roles = fetch_roles(guild_id)
        
        with col2:
            if not roles:
                st.warning("⚠️ Nenhum cargo encontrado.")
                selected_role = None
            else:
                selected_role = st.selectbox("Selecione o Cargo (Turma)", options=roles, format_func=lambda r: r["name"])
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
                categories = fetch_categories(guild_id)
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
                            res = create_category(guild_id, new_cat_name.strip(), int(new_cat_pos))
                        if res["success"]:
                            st.session_state.auto_select_category_id = res["data"]["id"]
                            st.success(f"Categoria '{new_cat_name}' criada com sucesso!")
                            time.sleep(1)
                            st.rerun()
                        else:
                            st.error(f"Erro ao criar categoria: {res.get('error')}")
            
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
                        
                        result = provision_channels(guild_id, category_id, role_id)
                        
                        if result["success"]:
                            st.session_state.provision_metrics = result["data"]["metrics"]
                            status.update(label="🎉 Provisionamento Concluído!", state="complete", expanded=False)
                        else:
                            status.update(label="❌ Erro no Processamento!", state="error", expanded=True)
                            st.error(result["error"])
                
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
