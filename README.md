# Clima por CEP com OpenTelemetry e Zipkin

Projeto desenvolvido para o desafio técnico do Go Expert. O sistema é composto por dois microsserviços em Go (Serviço A e Serviço B) que consultam o clima a partir de um CEP e utilizam OpenTelemetry e Zipkin para tracing distribuído.

---

## Como executar

### 1. Variáveis de ambiente
Crie um arquivo `.env` a partir do exemplo e informe sua chave da [WeatherAPI](https://www.weatherapi.com/):

```bash
cp .env.example .env
```

No `.env`:
```env
WEATHER_API_KEY=sua_chave_aqui
COMPOSE_BAKE=false
```

### 2. Subir o ambiente com Docker Compose
```bash
docker compose up --build
```

Endpoints disponíveis:
- Serviço A: http://localhost:8080
- Serviço B: http://localhost:8081
- Zipkin: http://localhost:9411
- OTel Collector: portas 4317 (gRPC) e 4318 (HTTP)

---

## Como testar

As requisições devem ser feitas no **Serviço A**, que valida o CEP (deve ter exatamente 8 dígitos e ser enviado como string) e repassa a chamada para o **Serviço B**.

### 1. Sucesso (200 OK)
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "01001000"}'
```

Resposta:
```json
{
  "city": "São Paulo",
  "temp_C": 28.5,
  "temp_F": 83.3,
  "temp_K": 301.65
}
```

### 2. CEP com formato inválido (422)
Se o CEP não for string ou não contiver 8 dígitos:
```bash
curl -i -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "1234567"}'
```

Resposta:
```
invalid zipcode
```

### 3. CEP não encontrado (404)
Se o formato for válido, mas o CEP não existir:
```bash
curl -i -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"cep": "99999999"}'
```

Resposta:
```
can not find zipcode
```

Se preferir testar pelo VS Code, as requisições também estão prontas em `test/api.http`.

---

## Visualização no Zipkin

1. Abra http://localhost:9411 no navegador.
2. Clique em **Run Query** (ou filtre pelo serviço `service-a`).
3. Clique no trace para ver o rastreamento completo:
   - Requisição partindo do `service-a` e propagando o contexto até o `service-b`.
   - Span `busca-cep`: mede a chamada externa ao ViaCEP.
   - Span `busca-temperatura`: mede a chamada externa à WeatherAPI.

---

## Testes

Para rodar os testes unitários:

```bash
go test -v ./...
```
