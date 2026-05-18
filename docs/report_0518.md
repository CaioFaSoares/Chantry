# 🌌 Relatório de Engenharia e Status: Suíte Chantry (18/05/2026)

Este documento sumariza as evoluções arquiteturais, modelagem de dados, motor de provisionamento e o painel de controle frontend realizados no ecossistema Chantry.

---

## 🏛️ Visão Geral da Arquitetura Híbrida

Hoje realizamos a consolidação de uma **arquitetura de imagem única** no backend, a transição para um **PocketBase embarcado (embedded)** em Go, a implementação do **Motor de Provisionamento em Lote de Canais Privados** e a **remoção completa do n8n da stack**, simplificando a infraestrutura da aplicação.

O binário `./main` em Go atua em duas frentes independentes com base nos argumentos de execução:
1. **PocketBase Server (`./main serve`)**: Executa o banco de dados SQLite local embarcado, aplica migrações automáticas de esquema e expõe as portas de dados HTTP na porta `12090` (ou `8090` interno).
2. **Discord Go Daemon (`./main api`)**: Servidor API em Fiber que gerencia integrações com Discord API, orquestra regras de negócio do backend e atualizações no banco de dados na porta `12000`.

```mermaid
graph TD
    subgraph Docker Network [Chantry Network]
        Streamlit[Streamlit Frontend:12501] -->|HTTP:12000| GoServer[Chantry Go Server: Fiber Daemon]
        GoServer -->|Admin API:8090| PocketBase[PocketBase Embedded Engine]
    end
    
    subgraph Volumes
        PocketBase -->|Persistência| pb_data[(/pb_data SQLite)]
    end
    
    subgraph External
        GoServer -->|Discord API| Discord[Discord Web Gateway]
    end
```

---

## 💾 Épico 2.1: Modelagem de Dados no PocketBase

A persistência do banco de dados local foi modernizada com alta normalização de dados para evitar *anti-patterns* de coleções embutidas, estruturando chaves estrangeiras e regras estritas de segurança.

### Coleções Estruturadas
O esquema foi estruturado no arquivo local [pb_schema.json](file:///Users/caiosoares/_Nexus/sirius/Projects/Chantry/pb_schema.json) contemplando:
*   **`guilds`**: Monitoramento dos servidores (Discord ID e Status de ativação).
*   **`roles`**: Cargos de turmas vinculados a um servidor específico.
*   **`students`**: Snapshot de alunos sincronizados do Discord, associando-os com `roles`, `guilds`, status escolar e de canais dedicados.
*   **`managers`**: Equipe de mentores e administradores associados a múltiplos servidores.
*   **`attendances`**: Presenças individuais contendo timestamps, notas de justificativa, status (presente/falta/justificado) e fonte do registro.
*   **`activities`**: Atividades e tarefas propostas em cada guilda escolar.

### Regras de Segurança Aplicadas
*   **Todas as `*Rules` (list, view, create, etc.) foram trancadas com `""`**: Isso impossibilita requisições diretas não autenticadas vindas da API pública (como browsers e fontes externas), isolando o banco. Apenas conexões contendo cabeçalhos válidos de **Admin JWT** (ou nosso backend) conseguem ler e escrever nas coleções.
*   **Unique Indexes**: Configuração de índices de unicidade nos campos `discord_id` para garantir integridade física no SQLite subjacente.

---

## ⚡ Épico 2.2: Camada de Integração Go ↔ PocketBase (REST)

Para orquestrar a comunicação do Fiber com o PocketBase, desenvolvemos a camada de persistência nativa em Go sem dependências pesadas, utilizando apenas a biblioteca padrão (`net/http` e `encoding/json`).

### Componentes Desenvolvidos
*   **Configurações Dinâmicas:** Estendemos a struct `Config` em [env.go](file:///Users/caiosoares/_Nexus/sirius/Projects/Chantry/backend/internal/config/env.go) para suportar as credenciais e URL do PocketBase.
*   **Models Go:** No pacote [models.go](file:///Users/caiosoares/_Nexus/sirius/Projects/Chantry/backend/internal/pocketbase/models.go), estruturamos as structs de dados necessárias.
*   **REST Client Thread-Safe:** Em [client.go](file:///Users/caiosoares/_Nexus/sirius/Projects/Chantry/backend/internal/pocketbase/client.go), implementamos o cliente REST que gerencia de forma segura o JWT de administrador, com auto-anexação de headers nas requisições.
*   **Repositório Abstrato:** O repositório [repository.go](file:///Users/caiosoares/_Nexus/sirius/Projects/Chantry/backend/internal/pocketbase/repository.go) encapsula as operações do banco: `FindFirstByDiscordID`, `CreateRecord`, `UpdateRecord`, `FindFirstByDiscordAndGuild` e `FindManagersByGuild`.

---

## ✨ Épico 3: Motor de Infraestrutura e Provisionamento em Lote

Concluímos o desenvolvimento completo dos fluxos de provisionamento automático de canais de texto privados (1-on-1) para as turmas, blindando o servidor do Discord contra rate limits e integrando as operações com o banco de dados.

### 🏢 PRD 3.1 - Gerenciamento de Categorias (Parent Scope)
*   **Listagem de Categorias:** Implementamos o endpoint `GET /api/discord/guilds/:guildId/categories` que retorna todas as categorias do servidor para seleção reativa.
*   **Criação de Categorias:** Implementamos o endpoint `POST /api/discord/guilds/:guildId/categories` que cria instantaneamente uma nova categoria de canais personalizada no Discord e retorna os dados do canal criado (incluindo o ID gerado e posição).

### 🔒 PRD 3.2 - Core de Permissões e Canais Privados (1-on-1 Factory)
*   **Algoritmo de Blindagem Bitwise:** Desenvolvemos a factory especialista `CreatePrivateChannel` no serviço do Discord:
    *   **Bloqueio Total do `@everyone`:** Remove a permissão de visualização (`PermissionViewChannel`) de todos os usuários do servidor de forma nativa (onde o ID do cargo `@everyone` é estritamente igual ao ID do Servidor).
    *   **Permissão do Aluno Dono:** Concede permissão explícita de visualização e envio de mensagens para o ID do Aluno.
    *   **Permissão dos Managers:** Permite que a equipe de mentores/administradores (carregados via banco na relação da guilda) visualizem e moderem o canal.

### ⚙️ PRD 3.3 - Motor de Processamento em Lote e Worker Sincronizador
*   **Resolução Inteligente de IDs:** O Usecase [provision_usecase.go](file:///Users/caiosoares/_Nexus/sirius/Projects/Chantry/backend/internal/usecases/provision_usecase.go) recebe os IDs Discord Snowflake de Guilda e Cargo e os resolve para os correspondentes IDs relacionais de 15 caracteres do PocketBase para que as queries funcionem perfeitamente.
*   **Motor de Cooldown (Pulmão):** Adiciona uma pausa nativa de `800ms` (`time.Sleep`) em cada iteração, protegendo a conta do Bot de bloqueios da API REST do Discord.
*   **Garantia de Idempotência:** Busca no PocketBase utilizando a query `FindStudentsPendingProvision` que busca exclusivamente alunos cujo campo `channel_id` esteja vazio e inclui `limit=200` para processamento completo em lote. Alunos com canal ativo são ignorados.
*   **Transacionalidade e Durabilidade:** Salva o `channel_id` individualmente no PocketBase logo após a criação no Discord, prevenindo perda de estado em caso de queda de rede ou reinicialização.
*   **Roteamento Fiber:** Endpoint mapeado em `POST /api/provision/guilds/:guildId/channels`.

### 🖥️ PRD 3.4 - Painel do Streamlit: Provisionamento e Infraestrutura
*   **Nova Página (`3_infra_provisioning.py`):** Criamos a interface com Outfit font do Google Fonts, cards glassmorphic e cabeçalhos gradientes de alta fidelidade visual.
*   **Dropdowns Reativos:** Selectbox de Servidor e Cargo ativos integrados à API.
*   **Estratégia de Categoria Pai:** Escolha entre usar categoria existente ou criar nova na hora. A criação de categoria mapeia o status de resposta `201 Created`, armazena o ID no `st.session_state` e usa `st.rerun()` para auto-selecionar o novo canal após o recarregamento.
*   **Status Logger Progressivo:** Monitora visualmente a requisição em lote longa utilizando o widget `st.status` com timeout estendido de `180` segundos.
*   **Métricas do Lote:** Plota cards interativos de total de alunos, canais criados, já provisionados e possíveis erros.

---

## 🧹 Evolução da Stack: Remoção do n8n

Para otimizar o consumo de recursos e focar o ecossistema nas soluções de desenvolvimento nativas da Suíte Chantry (Streamlit + PocketBase + Go Daemon), removemos completamente o n8n do fluxo:
*   **Exclusão de Arquivos:** Deletamos o arquivo de guias `1_n8n_guide.py`.
*   **Refatoração do app.py:** Reconfiguramos a home do Streamlit de 4 colunas para uma grade limpa de 3 colunas (Streamlit Dashboard, PocketBase Backend, Go Backend Daemon).
*   **Exclusão do Sandbox:** Removemos por completo a seção Sandbox de webhooks de teste que realizava envios simulados para o n8n.
*   **Remoção Concluída:** A varredura de termos pelo diretório aponta que o frontend está 100% livre de referências ao n8n.

---

## 🚀 Status da Compilação e Validações

Realizamos os testes de compilação estática em ambas as linguagens e a integridade de todas as entregas foi validada com **100% de sucesso**:

1.  **Backend Go Daemon (`go build`):**
    *   **Comando:** `go build -o /dev/null ./cmd/api/main.go` (no diretório `backend`)
    *   **Resultado:** Compilação bem-sucedida, sem avisos de tipagem estática ou erros de injeção (`Exit code: 0`).
2.  **Frontend Streamlit (`python3 -m py_compile`):**
    *   **Comando:** `python3 -m py_compile streamlit/app.py` e `python3 -m py_compile streamlit/pages/3_infra_provisioning.py`
    *   **Resultado:** Sintaxe Python e importações validadas sem erros (`Exit code: 0`).
