# Prompt para Devin

**Objetivo:** Atuar como Principal Software Engineer e Arquiteto de Soluções para mapear o ecossistema legado e desenhar uma nova arquitetura disruptiva, elástica e orientada a metadados para a plataforma.

**Contexto do Domínio:**

Você tem acesso aos repositórios que compõem o produto de crédito consignado CLT, especificamente o ecossistema responsável pela liquidação (baixa) e ressarcimento de parcelas. Atualmente, o processamento de milhares de pagamentos mensais (com crescimento de 4,4% a.m.) é orquestrado de forma estática por Step Functions, ECS e Lambdas na AWS. As regras de validação (códigos de motivo como 00, h8, bi, h3) e decisões de negócio (baixa da parcela mais antiga, baixa integral, parcial, ressarcimento) estão acopladas na infraestrutura e no código.

**A Missão:**

Nossa meta é modernizar essa plataforma para uma arquitetura maleável, dinâmica e anticorruptiva. A nova fundação técnica será baseada em:

1. Uma Anti-Corruption Layer (ACL) universal usando GraphQL em Go.
2. Motor de regras de negócio desacoplado utilizando CEL (Common Expression Language).
3. Orquestração dinâmica descentralizada baseada no padrão "Routing Slip" com Event Sourcing.
4. Observabilidade avançada e explicabilidade contínua com alta cardinalidade via Datadog.

Por favor, execute as seguintes tarefas em etapas sequenciais. Não avance para a próxima etapa sem antes concluir e documentar a anterior.

#### Etapa 1: Descoberta e Mapeamento do Estado Atual (As-Is)

Analise o código-fonte, configurações de infraestrutura (Terraform/CloudFormation) e documentações presentes nos repositórios para realizar a engenharia reversa do sistema atual.

- **Mapeamento de Domínio:** Identifique e liste todos os domínios, subdomínios, entidades principais (agrupando seus dados logicamente) e agregados envolvidos no processo de liquidação e ressarcimento.
    
- **Mapeamento Funcional e Regras de Negócio:** Extraia a árvore de decisão atual. Mapeie exatamente onde e como as lógicas de códigos de retorno (00, h8, bi, h3, etc.) e ações (baixa parcial, integral, etc.) estão implementadas.
    
- **Topologia de Integração:** Liste todas as conexões externas (APIs de terceiros, sistemas legados) e bases de dados (DynamoDB, RDS, etc.).
    
- **Entrega da Etapa 1:** Gere um documento Markdown detalhado e exporte diagramas de contexto e dependências (use Mermaid ou PlantUML) que possam ser facilmente integrados em plataformas de visualização arquitetural e grafos de sistemas (como o Flowbridge).
    

#### Etapa 2: Desenho da Arquitetura Alvo (To-Be)

Com base no mapeamento da Etapa 1, projete a nova arquitetura disruptiva baseada nos pilares definidos.

- **Desenho da ACL (GraphQL + Go):** Especifique como os _resolvers_ do GraphQL abstrairão a comunicação com as fontes de dados mapeadas na Etapa 1, servindo como o ponto único de entrada de dados para os workers.
    
- **Catálogo de Regras (CEL):** Transforme as regras de negócio mapeadas (As-Is) em exemplos práticos de scripts CEL. Especifique como o serviço Go irá cachear e avaliar essas expressões contra o _payload_ de entrada.
    
- **Motor de Roteamento (Routing Slip):** Desenhe o fluxo de vida de um evento. Defina a estrutura JSON do _routing slip_, como o worker avaliador injetará as etapas após processar o CEL, e como os workers de domínio (Lambdas/ECS) consumirão e repassarão o evento no barramento (EventBridge/Kafka).
    
- **Design de Telemetria e Explicabilidade:** Defina os pontos exatos de instrumentação no código Go para injeção de tags de negócios, garantindo que o ciclo de vida completo do _routing slip_ de uma parcela possa ser rastreado visualmente através das métricas de alta cardinalidade.
    
- **Entrega da Etapa 2:** Documentação técnica contendo diagramas de arquitetura e sequência (C4 Model) e a definição dos contratos de dados (JSON schemas do evento e schemas GraphQL).
    

#### Etapa 3: Estratégia de Reordenamento e Migração

Proponha um plano de transição tático e seguro, considerando o volume e o crescimento da operação.

- **Decoupling Progressivo:** Como podemos fatiar o monolito distribuído atual (Step Functions)? Proponha uma abordagem (ex: Strangler Fig Pattern) para rotear gradativamente percentuais específicos de tráfego para a nova infraestrutura.
    
- **Shadow Mode e Benchmarking:** Desenhe uma estratégia para rodar o motor CEL em paralelo (shadow mode) com as Step Functions atuais, apenas para comparar resultados, logs de explicabilidade e performance do Go antes da virada oficial.
    
- **Entrega da Etapa 3:** Um roadmap técnico detalhando fases, riscos mapeados, estratégias de _rollback_ por etapa e pré-requisitos de infraestrutura AWS a serem provisionados.
    

---

### Como gerenciar essa execução

Ao passar esse prompt para o Devin, recomendo que você o acompanhe de perto durante a transição entre as etapas. Como ele tem acesso aos repositórios, a **Etapa 1** será o momento onde ele mais fará buscas, leitura de arquivos e entendimento do ecossistema.

Quando ele gerar os diagramas em Mermaid/PlantUML na Etapa 1, você já terá insumos excelentes para plugar nas suas ferramentas de visualização e documentação Docusaurus.

A **Etapa 2** é onde a mágica acontece. O Devin pegará o emaranhado de lógicas da Step Function e vai convertê-las em sugestões de scripts CEL claros e desacoplados.

Quer que eu refine algum ponto específico desse prompt, talvez detalhando mais a parte de observabilidade ou a mecânica de eventos na AWS?