# 🌌 Chantry Orchestration Platform

O **Chantry** é uma plataforma de orquestração projetada para gerenciar comunidades no Discord, com foco especial em programas educacionais, mentorias ou turmas de treinamento. Ele automatiza fluxos de presença, provisionamento de infraestrutura (canais privativos) e comunicação em massa.

## 🚀 Tecnologias Core

*   **Go Server Daemon (`go-server`):** O motor de alta performance que gerencia a lógica de negócio, interações com a API do Discord, processamento de Cronjobs e sincronização de dados. O binário é "dual-purpose": pode atuar tanto como o servidor de API (Fiber) quanto como o servidor PocketBase, dependendo dos argumentos de inicialização.
*   **App:** Interface administrativa para monitoramento de métricas, configuração de horários e execução de comandos manuais.
*   **PocketBase:** Server-as-a-service leve (SQLite) que gerencia persistência de dados, autenticação e coleções (Estudantes, Presenças, Roles, etc). No Chantry, o PocketBase é estendido nativamente via código Go para suportar migrações automáticas.
*   **Discord Integration:** Integração nativa via WebSocket Gateway e REST API para automação de canais e interações via botões.

---

## 🏗️ Estrutura do Projeto

```text
Chantry/
├── server/                # Core da aplicação em Go
│   ├── cmd/api/           # Entrypoint da API Fiber
│   └── internal/
│       ├── config/        # Carregamento de variáveis de ambiente
│       ├── cron/          # Workers de broadcast e agendadores de presença
│       ├── discord/       # Cliente Discord e handlers de interação (Botões)
│       ├── handlers/      # Handlers HTTP para integração com o App
│       ├── migrations/    # Scripts de atualização do schema PocketBase
│       ├── pocketbase/    # Cliente e modelos do banco de dados
│       └── usecases/      # Lógica de negócio (Attendance, Sync, Provision)
├── db/                    # Banco de dados SQLite e logs do PocketBase (Persistente)
├── app/                   # Dashboard administrativo em Python
│   ├── app.py             # Página inicial e visão geral
│   ├── pages/             # Módulos específicos (Sync, Provisioning, Attendance, etc)
│   └── utils/             # Clientes de API para comunicação com o server
└── docker-compose.yml     # Orquestrador da infraestrutura (Go, App, PocketBase)
```

---

## 🛠️ Funcionalidades Principais

1.  **Presença Automatizada (Attendance):**
    *   Envio automático de prompts de Check-in/Check-out via Discord com base em cronogramas.
    *   Monitoramento de status em tempo real (Pendente, Em progresso, Concluído).
    *   Gestão de períodos de *cooldown* para estudantes.
2.  **Provisionamento de Infraestrutura:**
    *   Criação em lote de canais privativos (1-on-1) para estudantes e mentores.
    *   Configuração automática de permissões e boas-vindas.
3.  **Sincronização de Dados:**
    *   Sincronização bidirecional de membros e cargos (roles) entre Discord e PocketBase.
    *   Mapeamento de estudantes por ID de Discord.
4.  **Broadcast Center:**
    *   Envio de mensagens agendadas or imediatas para canais ou cargos específicos.

---

## ⚡ Como Rodar

### 1. Requisitos
*   Docker e Docker Compose instalados.
*   Um bot no Discord configurado com as devidas permissões (Administrator recomendado para POC).

### 2. Configuração (Variáveis de Ambiente)
Crie um arquivo `.env` na raiz do projeto com as seguintes chaves:

```env
PB_ADMIN_EMAIL=admin@example.com
PB_ADMIN_PASSWORD=password123
DISCORD_APP_ID=seu_app_id
DISCORD_PUBLIC_KEY=sua_public_key
DISCORD_BOT_TOKEN=seu_bot_token
```

### 3. Execução
Suba o ambiente completo:
```bash
docker compose up --build
```

### 4. Portas dos Serviços
*   💻 **App Dashboard:** [http://localhost:12501](http://localhost:12501)
*   🗄️ **PocketBase Admin:** [http://localhost:12090/_/](http://localhost:12090/_/)
*   🐹 **Go Server API:** [http://localhost:12000](http://localhost:12000)

---

## 📂 Persistência e Desenvolvimento
O projeto utiliza *bind mounts* para as pastas `db` e `app`. Isso permite que:
1.  **Dados do Banco:** Configurações e registros do PocketBase sejam persistidos entre reinicializações.
2.  **Hot-Reload:** Alterações no código do App sejam refletidas instantaneamente sem necessidade de rebuild do container.
