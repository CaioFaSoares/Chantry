import streamlit as st

st.set_page_config(
    page_title="Guia n8n - Chantry Suite",
    page_icon="🔌",
    layout="wide"
)

# Reuse the premium CSS style for cards, badges, and headers
custom_css = """
<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap');

html, body, [class*="css"], .stMarkdown {
    font-family: 'Outfit', sans-serif !important;
}

.subpage-header {
    background: linear-gradient(135deg, #EC4899 0%, #A855F7 50%, #6366F1 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    font-weight: 700;
    font-size: 2.5rem;
    margin-bottom: 0.5rem;
    letter-spacing: -0.5px;
}

.step-card {
    background: rgba(30, 41, 59, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 15px;
    backdrop-filter: blur(10px);
}

.step-num {
    background: linear-gradient(135deg, #6366F1 0%, #EC4899 100%);
    color: white;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    margin-right: 10px;
}

.step-title {
    font-size: 1.15rem;
    font-weight: 600;
    color: #F8FAFC;
    display: inline-flex;
    align-items: center;
}
</style>
"""
st.markdown(custom_css, unsafe_allow_html=True)

st.markdown("<h1 class='subpage-header'>🔌 Guia: Conectando Streamlit ao n8n</h1>", unsafe_allow_html=True)
st.write("Aprenda a estruturar fluxos no n8n e dispará-los a partir da sua interface Streamlit em 3 passos simples.")

st.markdown("---")

col1, col2 = st.columns([3, 2])

with col1:
    st.markdown("### 🛠️ Fluxo de Trabalho Recomendado")
    
    # Step 1
    st.markdown("""
    <div class='step-card'>
        <div class='step-title'><span class='step-num'>1</span> Criar o Nó de Webhook no n8n</div>
        <p style='color: #94A3B8; margin-top: 8px; font-size: 0.95rem;'>
            Abra o painel do n8n em <a href="http://localhost:5678" target="_blank" style="color:#6366F1;">localhost:5678</a>, crie um novo workflow e adicione um nó do tipo <b>Webhook</b>.<br>
            Defina as configurações:<br>
            • <b>HTTP Method:</b> <code>POST</code><br>
            • <b>Path:</b> Escolha um identificador único (ex: <code>minha-poc-api</code>)<br>
            • <b>Response Mode:</b> Mude para <code>When Last Node Finishes</code> se você quiser que o Streamlit receba a resposta final processada.
        </p>
    </div>
    """, unsafe_allow_html=True)
    
    # Step 2
    st.markdown("""
    <div class='step-card'>
        <div class='step-title'><span class='step-num'>2</span> Usar a URL de Teste (*Test URL*)</div>
        <p style='color: #94A3B8; margin-top: 8px; font-size: 0.95rem;'>
            O n8n possui dois modos de execução:<br>
            • <b>Test URL (Recomendado para POCs):</b> Usado enquanto você está desenhando o fluxo e com a tela aberta no n8n. Excelente para debugar e ver os dados entrando em tempo real.<br>
            • <b>Production URL:</b> Usada apenas depois que você ativa o fluxo no interruptor <i>Active</i> do topo direito.
        </p>
    </div>
    """, unsafe_allow_html=True)

    # Step 3
    st.markdown("""
    <div class='step-card'>
        <div class='step-title'><span class='step-num'>3</span> Conectar o Processamento e Responder</div>
        <p style='color: #94A3B8; margin-top: 8px; font-size: 0.95rem;'>
            Conecte nós de transformação (Javascript, OpenAi, Supabase, Postgres) e finalize o fluxo com um nó de retorno (ou deixe o n8n retornar automaticamente o payload de saída do último nó).
        </p>
    </div>
    """, unsafe_allow_html=True)

with col2:
    st.markdown("### 💡 Boas Práticas para POCs")
    
    st.info("""
    🔒 **Segurança das Credenciais:**
    Como os dados do n8n estão na pasta `./n8n_data`, suas chaves de API criadas na interface ficam salvas localmente no arquivo SQLite. Não compartilhe essa pasta publicamente se adicionar tokens reais de produção.
    """)
    
    st.success("""
    ⚡ **Velocidade de Execução:**
    Sempre prefira enviar dados no formato **JSON** (`application/json`). É nativo tanto no Python (`requests.post(..., json=dados)`) quanto no n8n.
    """)
    
    st.warning("""
    🔄 **Hot Reload do Streamlit:**
    Como seu diretório `./streamlit` está montado no Docker, qualquer alteração de código que você salvar nestes arquivos atualizará o dashboard instantaneamente.
    """)

st.markdown("---")
st.markdown("### 📝 Exemplo de Payload Complexo")
st.write("Um padrão muito comum em automações do n8n é passar metadados e parâmetros estruturados:")

complex_json_example = """{
  "sender": "Streamlit App",
  "timestamp": "2026-05-18T09:43:00Z",
  "action": "orchestrate_pipeline",
  "parameters": {
    "dry_run": false,
    "limit_records": 100,
    "target_schema": "staging"
  },
  "records": [
    { "id": 1, "nome": "Caio Soares", "status": "active" },
    { "id": 2, "nome": "Antigravity Agent", "status": "pending" }
  ]
}"""

st.code(complex_json_example, language="json")
