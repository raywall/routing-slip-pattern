---
sidebar_position: 5
sidebar_label: "Capacidades e Benefícios"
---

# Capacidades e Benefícios

O modelo `Routing Slip` brilha em integrações complexas onde a confiabilidade do processo não pode ser comprometida pela instabilidade transitória de um provedor externo[cite: 6]. 

## Diferenciais Técnicos

| Capacidade Arquitetural | Valor Entregue |
| :--- | :--- |
| **Declaratividade em YAML** | A regra de negócio não fica engessada na compilação do Go. Ela se torna auditável para times de negócio e foca puramente no "O Quê"[cite: 6]. |
| **Idempotência Nativa** | O lock do *State Store* evita que um processamento que envia uma notificação ou movimenta dinheiro aconteça em duplicidade[cite: 6]. |
| **Enriquecimento Assíncrono** | Dados periféricos são apensados ao *payload* em memória antes do motor de regras avaliar o caso[cite: 6]. |
| **Composição de Scripts** | Fluxos titânicos podem ser fatiados em arquivos menores (`workflow_ref`), mantendo a rastreabilidade do `trace_id` original intacta[cite: 6]. |
| **Retomada Cirúrgica** | Um erro no passo 4 de 10 apenas pausa o fluxo. O *resume* recomeça diretamente do passo 4 com o *payload* do exato momento da falha[cite: 6]. |

## Estudo de Caso: Baixa de Parcela de Consignado

Para materializar o ganho, vamos imaginar um fluxo de **Baixa de Consignado CLT** ou liquidação financeira:

1. **Validação:** Um evento chega via fila informando o pagamento de uma parcela[cite: 6].
2. **Enriquecimento:** O *workflow* chama o GraphQL Connector para buscar os dados consolidados do contrato[cite: 6].
3. **Decisão:** Um step avalia as políticas da averbadora[cite: 6].
4. **Execução:** O desconto é aplicado no saldo devedor[cite: 6].
5. **Notificação:** O sistema tenta comunicar o averbador externo (INSS/Empresa Privada)[cite: 6].

**O problema resolvido:** Se a API do averbador externo estiver fora do ar no passo 5, o sistema *não* falha catastróficamente[cite: 6]. O State Store salva o cursor no passo 5. Quando o *retry* ocorrer mais tarde, a arquitetura do Routing Slip sabe que o passo 4 (o desconto financeiro) já foi processado e executa apenas a tentativa de comunicação, eliminando a criação de lógicas complexas de validação compensatória[cite: 6]. Além disso, como cada passo emite cardinalidade rica, o Datadog mostrará exatamente o gargalo na integração externa de forma visual.