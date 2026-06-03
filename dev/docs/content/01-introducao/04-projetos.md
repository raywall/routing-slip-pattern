---
sidebar_position: 4
sidebar_label: "Módulos do Ecossistema"
---

# Módulos do Ecossistema

Para garantir flexibilidade arquitetural, o framework é particionado em projetos independentes[cite: 5]. Essa topologia permite que você acople apenas as partes necessárias à realidade do seu *squad*.

## 1. Routing Slip Pattern (Runtime)

O coração da execução. Ele consome o YAML, escuta os gatilhos (REST, Kafka, SQS), orquestra a chamada sequencial dos *handlers* e interage com o *State Store*[cite: 5]. 
**Uso ideal:** Orquestração distribuída, necessidade de reprocessamento seguro a partir de falhas e padronização de lógicas repetitivas[cite: 5].


## 2. Go GraphQL Connector

Uma API unificada configurável que age como *BFF* (Backend for Frontend) ou fachada de integração. Ele expõe uma interface GraphQL capaz de resolver chamadas em bancos relacionais, DynamoDB, Redis ou APIs legadas[cite: 5].
**Uso ideal:** Concentrar a complexidade de autenticação, *timeouts* e mapeamento de dados externos, permitindo que o *workflow* apenas declare um step de `graphql_enrich` limpo[cite: 5].


## 3. Custom Business Metrics

Um módulo de ingestão para consolidar as métricas técnicas (tempo por handler) e funcionais (status das regras de negócio). Ele permite acompanhar as jornadas cruzando o `correlation_id`[cite: 5].
**Uso ideal:** Diagnóstico profundo de gargalos operacionais e monitoramento de *retries* sistemáticos[cite: 5].


## Execução Local (Modo Desenvolvedor)

Na raiz do *workspace*, o ecossistema disponibiliza comandos via `Makefile` para levantar os módulos[cite: 5]:

| Comando | Comportamento | Recomendação de Uso |
| :--- | :--- | :--- |
| `make prepare` | Inicia *containers* isolados, simulando fielmente a rede, bancos de dados (Dynamo local) e filas de uma infra real[cite: 5]. | **Padrão.** Testes integrados e validações de comportamento assíncrono rigorosas[cite: 5]. |
| `make run-compact` | Levanta o runtime, GraphQL e métricas compartilhando o mesmo contêiner (portas: 8088, 8090, etc.) com estado apenas em memória[cite: 5]. | Provas de conceito, demonstrações rápidas ou validações puras de script YAML[cite: 5]. |
