# 🌌 Chantry Orchestration Platform

Este é o ambiente inicial do projeto **Chantry**, estruturado para permitir prototipagem rápida e desenvolvimento ágil de Provas de Conceito (POCs) utilizando **Streamlit** para a interface interativa, **n8n** para automação/orquestração visual de fluxos e **PocketBase** como um banco de dados e backend em tempo real super leve.

Todos os serviços estão integrados via **Docker Compose** e configurados para total persistência de dados local no diretório do projeto.

---

## 🏗️ Estrutura do Projeto

```text
Chantry/
├── .gitignore             # Evita commitar caches, logs e arquivos temporários
├── README.md              # Este manual de instruções
├── docker-compose.yml     # Orquestrador dos containers Streamlit, n8n e PocketBase
├── n8n_data/              # [IMPORTANTE] SQLite e configurações do n8n (salvo localmente!)
├── pb_data/               # [IMPORTANTE] SQLite, configurações e uploads do PocketBase (salvo localmente!)
└── streamlit/             # Diretório do aplicativo Streamlit
    ├── .streamlit/
    │   └── config.toml    # Tema premium escuro (Deep Dark Indigo Theme)
    ├── Dockerfile         # Dockerfile de build para o Streamlit (Python 3.11-slim corrigido)
    ├── requirements.txt   # Dependências python (Pandas, Plotly, Requests, etc.)
    ├── app.py             # Landing Page, status unificado e Sandbox de webhooks
    └── pages/
        └── 1_n8n_guide.py # Guia dinâmico de integrações
```

---

## ⚡ Como Rodar o Ambiente com Docker Compose

### Requisitos:
* Docker Desktop ou OrbStack instalado e ativo no sistema.

### Passo a Passo:

1. Na raiz do projeto, suba os containers executando:
   ```bash
   docker compose up --build
   ```
2. Aguarde o download das imagens e a compilação local da imagem do Streamlit.
3. Acesse os serviços nos links abaixo:
   * 💻 **Streamlit Dashboard:** [http://localhost:8501](http://localhost:8501)
   * ⚙️ **n8n Automation Engine:** [http://localhost:5678](http://localhost:5678)
   * 🗄️ **PocketBase Backend:** [http://localhost:8090/_/](http://localhost:8090/_/) (Painel Admin)

---

## 📂 Persistência de Dados e Versionamento Local

Os serviços do **n8n** e do **PocketBase** estão configurados para apontar para pastas locais (`./n8n_data` e `./pb_data`) através de *bind mounts*. Isso traz duas vantagens fundamentais para o desenvolvimento do Chantry:
1. **Segurança contra perda:** Seus fluxos no n8n, esquemas de tabelas, registros, autenticações e arquivos salvos no PocketBase não serão perdidos mesmo que os containers sejam reconstruídos.
2. **Versionamento no Git:** Toda a configuração, esquemas de tabelas e bancos SQLite (`database.sqlite`, `data.db`, `api.db`) ficam armazenados diretamente no repositório local. Desta forma, ao dar `git commit`, todas as suas POCs de dados, APIs e fluxos estarão salvos no histórico do projeto.

---

## 💻 Desenvolvimento Local do Streamlit (Opcional / Fallback)

Se você preferir executar o **Streamlit localmente** fora do Docker (por exemplo, se o daemon do Docker estiver desligado), é possível rodar usando um ambiente virtual Python:

1. Navegue para o diretório raiz e crie um ambiente virtual:
   ```bash
   python3 -m venv .venv
   ```
2. Ative o ambiente virtual:
   ```bash
   source .venv/bin/activate
   ```
3. Instale as dependências:
   ```bash
   pip install -r streamlit/requirements.txt
   ```
4. Inicie o Streamlit:
   ```bash
   streamlit run streamlit/app.py
   ```
   O console local abrirá a página em [http://localhost:8501](http://localhost:8501).

---

## 🧬 Hot-Reloading Ativo

Tanto rodando pelo Docker quanto rodando localmente, o código do Streamlit suporta **Hot-Reload**. Ou seja: qualquer mudança que você fizer e salvar nos arquivos dentro de `./streamlit/` será detectada pelo Streamlit e você poderá recarregar a interface com um clique no navegador sem precisar reiniciar o servidor!
