# 🌌 Chantry Orchestration Platform

Este é o ambiente inicial do projeto **Chantry**, estruturado para permitir prototipagem rápida e desenvolvimento ágil de Provas de Conceito (POCs). O Chantry utiliza:
* **Streamlit:** Interface interativa e dinâmica para visualizações e sandbox de comandos.
* **Go Backend Daemon (`go-server`):** O "cérebro" de alta performance da aplicação, responsável por conexões nativas com a API do Discord, processamento de Cronjobs e sincronizações de dados.
* **n8n:** Orquestrador visual de fluxos de automações e webhooks.
* **PocketBase:** Banco de dados relacional (SQLite) e autenticação em tempo real super leve.

Todos os serviços estão integrados via **Docker Compose** e configurados em uma faixa de **portas isoladas (série `12XXX`)** para garantir conflito zero com qualquer outro projeto local.

---

## 🏗️ Estrutura do Projeto

```text
Chantry/
├── .gitignore             # Evita commitar caches, logs, binários Go e journals de banco
├── README.md              # Este manual de instruções
├── docker-compose.yml     # Orquestrador dos containers Streamlit, Go, n8n e PocketBase
├── n8n_data/              # [IMPORTANTE] SQLite e configurações persistentes do n8n
├── pb_data/               # [IMPORTANTE] SQLite, tabelas e uploads do PocketBase
├── backend/               # [NOVO] Serviço backend de alta performance em Go
│   ├── cmd/
│   │   └── api/
│   │       └── main.go    # Entrypoint (servidor Fiber com Logger/CORS e Healthcheck)
│   ├── Dockerfile         # Dockerfile de compilação multi-stage (Docker Alpine)
│   └── go.mod             # Mod do Go module
└── streamlit/             # Diretório do aplicativo Streamlit (Frontend)
    ├── .streamlit/
    │   └── config.toml    # Tema premium escuro (Deep Dark Indigo Theme)
    ├── Dockerfile         # Dockerfile de build para o Streamlit (Python 3.11-slim corrigido)
    ├── requirements.txt   # Dependências python (Pandas, Plotly, Requests, etc.)
    ├── app.py             # Landing Page de 4 colunas e Sandbox de Webhooks
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
2. O Docker compilará automaticamente a imagem do Streamlit e a imagem do Go Backend (via build multi-stage, **sem necessitar de instalação local do Go ou dependências no seu host**).
3. Acesse os serviços locais através das portas da série `12XXX`:
   * 💻 **Streamlit Dashboard:** [http://localhost:12501](http://localhost:12501)
   * ⚙️ **n8n Automation Engine:** [http://localhost:12678](http://localhost:12678)
   * 🗄️ **PocketBase Admin Panel:** [http://localhost:12090/_/](http://localhost:12090/_/)
   * 🐹 **Go Backend Healthcheck:** [http://localhost:12000/api/health](http://localhost:12000/api/health)

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
4. Inicie o Streamlit localmente (observe que por padrão ele iniciará na porta `8501`, mas consumirá os serviços Docker que estiverem ativos):
   ```bash
   streamlit run streamlit/app.py
   ```
   O console local abrirá a página em [http://localhost:8501](http://localhost:8501).

---

## 🧬 Hot-Reloading Ativo

Tanto rodando pelo Docker quanto rodando localmente, o código do Streamlit suporta **Hot-Reload**. Ou seja: qualquer mudança que você fizer e salvar nos arquivos dentro de `./streamlit/` será detectada pelo Streamlit e você poderá recarregar a interface com um clique no navegador sem precisar reiniciar o servidor!
