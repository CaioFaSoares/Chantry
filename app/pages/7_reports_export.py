import streamlit as st
import pandas as pd
from datetime import datetime
from utils.api_client import fetch_guilds, fetch_guild_roles_config, fetch_export_report

st.set_page_config(page_title="Central de Relatórios BI", page_icon="📈", layout="wide")

st.markdown("""
    <style>
        .metric-card {
            background-color: #1E1E1E;
            border-radius: 10px;
            padding: 20px;
            text-align: center;
            border: 1px solid #333;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .metric-value {
            font-size: 2rem;
            font-weight: bold;
            color: #FFFFFF;
        }
        .metric-label {
            font-size: 0.9rem;
            color: #A0A0A0;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
    </style>
""", unsafe_allow_html=True)

st.title("📈 Central de Relatórios e BI")
st.write("Agregação de dados de engajamento e exportação de presenças (CSV).")

# ==========================================
# 1. Filtros de Seleção (Barra Superior)
# ==========================================
guilds = fetch_guilds()
if not guilds:
    st.warning("Nenhum servidor Discord configurado. Verifique a conexão com o Go Server.")
    st.stop()

with st.container():
    col1, col2, col3 = st.columns(3)
    
    with col1:
        guild_opts = {g['id']: g['name'] for g in guilds}
        selected_guild_id = st.selectbox(
            "Servidor (Guilda)", 
            options=list(guild_opts.keys()), 
            format_func=lambda x: guild_opts[x]
        )
        
    with col2:
        roles = fetch_guild_roles_config(guild_id=selected_guild_id)
        role_opts = {"all": "🌍 Todas as Turmas / Roles"}
        for r in roles:
            role_opts[r['discord_id']] = f"{r['name']} (Turno: {r['shift']})"
            
        selected_role_id = st.selectbox(
            "Filtrar por Turma", 
            options=list(role_opts.keys()), 
            format_func=lambda x: role_opts[x]
        )
        
    with col3:
        # Date input returns a tuple of 1 or 2 elements
        date_range = st.date_input("Selecione o Período", [])

st.markdown("---")

# ==========================================
# 2. Ação e Carregamento de Dados
# ==========================================
if st.button("📊 Gerar Relatório", type="primary"):
    if len(date_range) != 2:
        st.warning("⚠️ Por favor, selecione as datas de **início** e **fim** no campo 'Período' antes de gerar o relatório.")
    else:
        start_date_str = date_range[0].strftime("%Y-%m-%d")
        end_date_str = date_range[1].strftime("%Y-%m-%d")
        
        with st.spinner(f"Processando dados de {start_date_str} até {end_date_str}..."):
            report_data = fetch_export_report(selected_guild_id, start_date_str, end_date_str, selected_role_id)
            
            if not report_data or not report_data.get("records"):
                st.info(f"Nenhum registo de presença encontrado no período de {start_date_str} a {end_date_str}.")
            else:
                summary = report_data["summary"]
                records = report_data["records"]
                
                # ==========================================
                # 3. Painel de BI (Summary)
                # ==========================================
                st.subheader("Métricas do Período")
                m1, m2, m3, m4, m5 = st.columns(5)
                
                # Formula to format percentage safely
                rate = summary.get("attendance_rate_percent", 0.0)
                rate_color = "#4CAF50" if rate >= 75.0 else ("#FF9800" if rate >= 50 else "#F44336")
                
                m1.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: {rate_color};">{rate:.1f}%</div>
                    <div class="metric-label">Taxa de Presença</div>
                </div>
                """, unsafe_allow_html=True)
                
                m2.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value">{summary.get("total_records", 0)}</div>
                    <div class="metric-label">Total Esperado</div>
                </div>
                """, unsafe_allow_html=True)
                
                m3.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: #4CAF50;">{summary.get("total_presents", 0)}</div>
                    <div class="metric-label">Presenças Válidas</div>
                </div>
                """, unsafe_allow_html=True)
                
                m4.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: #FFC107;">{summary.get("total_lates", 0)}</div>
                    <div class="metric-label">Atrasos</div>
                </div>
                """, unsafe_allow_html=True)
                
                m5.markdown(f"""
                <div class="metric-card">
                    <div class="metric-value" style="color: #F44336;">{summary.get("total_absents", 0)}</div>
                    <div class="metric-label">Faltas</div>
                </div>
                """, unsafe_allow_html=True)
                
                st.markdown("<br>", unsafe_allow_html=True)
                
                # ==========================================
                # 4. Tabela de Dados e Exportação CSV
                # ==========================================
                st.subheader("Registos Detalhados")
                
                # Converte JSON para Pandas DataFrame
                df = pd.DataFrame(records)
                
                # Renomear colunas para PT-BR
                df = df.rename(columns={
                    "student_name": "Nome do Aluno",
                    "student_discord_id": "Discord ID",
                    "role_name": "Turma / Role",
                    "date": "Data",
                    "clock_in": "Entrada",
                    "clock_out": "Saída",
                    "status": "Status"
                })
                
                # Traduzir status
                status_map = {
                    "completed": "✅ Completo",
                    "late": "⏳ Atrasado",
                    "absent": "❌ Falta",
                    "justified": "📝 Justificado",
                    "pending_checkout": "🔄 Em Progresso"
                }
                df["Status"] = df["Status"].map(lambda x: status_map.get(x, x))
                
                # Exibir DataFrame na UI
                st.dataframe(df, use_container_width=True, hide_index=True)
                
                # Gerar CSV na Memória
                csv_bytes = df.to_csv(index=False).encode('utf-8')
                
                # Botão Nativo de Download
                st.download_button(
                    label="📥 Descarregar Folha de Presenças (CSV)",
                    data=csv_bytes,
                    file_name=f"relatorio_presencas_{start_date_str}_a_{end_date_str}.csv",
                    mime="text/csv",
                    type="primary"
                )
