# Clima por CEP com Observabilidade (OpenTelemetry + Zipkin) — Go Expert

Projeto desenvolvido como parte do desafio técnico da pós-graduação **Go Expert**. O sistema é composto por uma arquitetura distribuída com dois microsserviços em Go (**Serviço A** e **Serviço B**) que cooperam para consultar a temperatura de uma cidade baseada no CEP, instrumentados com **OpenTelemetry (OTEL)** e **Zipkin** para rastreamento distribuído (*Distributed Tracing*).

---

## 🏛️ Arquitetura do Sistema

```
                      +-------------------+
                      |   Cliente HTTP    |
                      +---------+---------+
                                |
                    POST /      | (JSON: {"cep": "..."})
                                v
                      +-------------------+
                      |     Serviço A     |  (Porta 8080)
                      |  Input & Valida   |
                      +---------+---------+
                                |
             W3C TraceContext   | POST /
             (traceparent)      v
                      +-------------------+
                      |     Serviço B     |  (Porta 8081)
                      |   Orquestração    |
                      +---+-----------+---+
                          |           |
             Span: busca-cep          | Span: busca-temperatura
                          v           v
                     [ ViaCEP ]   [ WeatherAPI ]
                          |           |
                          +-----+-----+
                                |
                                v (Spans OTLP gRPC :4317)
                      +-------------------+
                      |   OTel Collector  |  (:4317 / :4318)
                      +---------+---------+
                                |
                                v (:9411/api/v2/spans)
                      +-------------------+
                      |      Zipkin       |  (UI :9411)
                      +-------------------+
```

### Componentes:
1. **Serviço A (Input - Porta 8080):** Porta de entrada HTTP. Recebe a requisição via `POST`, valida se o CEP possui exatamente 8 dígitos numéricos como string e encaminha a requisição para o Serviço B injetando o contexto de rastreamento distribuído (**W3C TraceContext**).
2. **Serviço B (Orquestração - Porta 8081):** Extrai o contexto de tracing, consulta o **ViaCEP** para obter a localidade (com span manual `busca-cep`), consulta a **WeatherAPI** para obter a temperatura (com span manual `busca-temperatura`), calcula as conversões (Celsius, Fahrenheit, Kelvin) e retorna a resposta formatada.
3. **OTel Collector (Portas 4317 / 4318):** Recebe os spans via protocolo OTLP gRPC dos serviços A e B e exporta para o Zipkin.
4. **Zipkin (Porta 9411):** Interface visual para consulta e auditoria dos traços distribuídos.

---

## 📋 Especificações da API

### Serviço A (Porta 8080)

* **Método:** `POST`
* **Rota:** `/` (também aceita `/cep` ou `/clima`)
* **Headers:** `Content-Type: application/json`
* **Payload:**
  ```json
  {
    "cep": "01001000"
  }
  ```

#### Respostas:
| Status Code | Cenário | Resposta |
| :--- | :--- | :--- |
| **`200 OK`** | Sucesso | `{"city": "São Paulo", "temp_C": 28.5, "temp_F": 83.3, "temp_K": 301.65}` |
| **`422 Unprocessable Entity`** | CEP com formato inválido (≠ 8 dígitos, letras, caracteres especiais) | `invalid zipcode` |
| **`404 Not Found`** | CEP válido, mas inexistente no ViaCEP | `can not find zipcode` |

---

## 🚀 Como Executar o Projeto

### Pré-requisitos
* [Docker](https://www.docker.com/) e [Docker Compose](https://docs.docker.com/compose/) instalados
* Chave gratuita da [WeatherAPI](https://www.weatherapi.com/)

---

### 1. Configurar as Variáveis de Ambiente
Crie um arquivo `.env` na raiz do projeto baseado no `.env.example`:

```bash
cp .env.example .env
```

Edite o arquivo `.env` informando sua chave:
```env
WEATHER_API_KEY=sua_chave_weatherapi_aqui
```

---

### 2. Iniciar Todos os Serviços via Docker Compose
Execute o comando na raiz do projeto:

```bash
docker compose up --build
```

O Docker Compose inicializará os 4 containers:
* **Serviço A:** `http://localhost:8080`
* **Serviço B:** `http://localhost:8081`
* **OTel Collector:** portas `4317` (gRPC) e `4318` (HTTP)
* **Zipkin UI:** `http://localhost:9411`

---

## 🧪 Como Realizar Requisições de Teste

### 1. Cenário de Sucesso (200 OK)
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "01001000"}'
```
**Resposta esperada (`200 OK`):**
```json
{
  "city": "São Paulo",
  "temp_C": 21.6,
  "temp_F": 70.88,
  "temp_K": 294.75
}
```

---

### 2. Cenário de CEP Inválido (422 Unprocessable Entity)
* Menos ou mais de 8 dígitos, letras ou hífen:
```bash
curl -i -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "1234567"}'
```
**Resposta esperada (`422 Unprocessable Entity`):**
```text
invalid zipcode
```

---

### 3. Cenário de CEP Inexistente (404 Not Found)
* 8 dígitos numéricos válidos, porém inexistente na base postal:
```bash
curl -i -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "99999999"}'
```
**Resposta esperada (`404 Not Found`):**
```text
can not find zipcode
```

---

## 🔍 Como Acessar o Zipkin e Visualizar os Traços

1. Abra o navegador em: **[http://localhost:9411](http://localhost:9411)**.
2. Na barra de pesquisa superior, clique no botão **RUN QUERY** (ou pesquise por `service-a` no campo *Service Name*).
3. Você verá a lista de requisições rastreadas.
4. Clique em um dos traços para abrir a árvore detalhada de spans. Você verá a cascata completa:
   * **`service-a-http`** / **`service-a-receive-request`** (tempo total no Serviço A)
   * Chamada HTTP para o Serviço B
   * **`service-b-http`** / **`service-b-orchestration`** (tempo total no Serviço B)
   * **`busca-cep`** *(span manual medindo a chamada externa ao ViaCEP)*
   * **`busca-temperatura`** *(span manual medindo a chamada externa à WeatherAPI)*

---

## 🧪 Testes Automatizados

O projeto conta com suíte de testes unitários isolados com mocks para validação de regras de CEP, conversões e fluxo HTTP.

### Executar testes localmente com Go:
```bash
go test -v ./...
```

### Executar testes via Docker (sem necessidade de Go instalado):
```bash
docker run --rm -v "$PWD:/app" -w /app golang:1.24-alpine go test -v ./...
```

---

## 📁 Estrutura de Arquivos

```text
.
├── cmd/
│   ├── service-a/
│   │   └── main.go                 # Entrada do Serviço A
│   └── service-b/
│       └── main.go                 # Entrada do Serviço B
├── config/
│   └── otel-collector-config.yaml  # Configuração do OpenTelemetry Collector
├── internal/
│   ├── entity/
│   │   ├── weather.go              # Entidades, conversões e validação de CEP
│   │   └── weather_test.go         # Testes de unidade das entidades
│   ├── servicea/
│   │   ├── handler.go              # Handler HTTP do Serviço A e injeção W3C
│   │   └── handler_test.go         # Testes do Serviço A
│   ├── serviceb/
│   │   ├── handler.go              # Handler HTTP do Serviço B e extração W3C
│   │   ├── viacep.go               # Integração ViaCEP com span manual "busca-cep"
│   │   ├── weatherapi.go           # Integração WeatherAPI com span manual "busca-temperatura"
│   │   └── service_test.go         # Testes do Serviço B
│   └── telemetry/
│       └── tracer.go               # Inicialização do OTel SDK com exporter gRPC OTLP
├── test/
│   └── api.http                    # Requisições prontas para testes no VS Code REST Client
├── .env.example                    # Exemplo de configuração de variáveis
├── .gitignore                      # Ignora credenciais, executáveis e arquivos temporários
├── docker-compose.yaml             # Orquestração do Service A, Service B, Collector e Zipkin
├── Dockerfile.service-a            # Multi-stage build enxuto do Serviço A
├── Dockerfile.service-b            # Multi-stage build enxuto do Serviço B
├── go.mod
├── go.sum
└── README.md
```
