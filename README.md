# Cupcake Store — Backend

API REST da loja virtual Cupcake Store, feita em Go com PostgreSQL. Cuida de cadastro, login, catálogo de cupcakes e pedidos.

> O front-end fica em outro repositório (`cupcake_store_frontend`) e roda em `http://localhost:5173`. Este projeto só expõe a API em `http://localhost:8080`.

## Pré-requisitos
- Go 1.22 ou mais novo
- PostgreSQL (ou Docker para subir um)
- `psql` para aplicar o schema e o seed

## Como rodar
1. Suba um PostgreSQL (exemplo com Docker, na porta 5433):
```bash
docker run --name cupcake_pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cupcake_store -p 5433:5432 -d postgres:16
```

2. Crie as tabelas e os cupcakes de exemplo:
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5433/cupcake_store?sslmode=disable"
psql "$DATABASE_URL" -f db/schema.sql
psql "$DATABASE_URL" -f db/seed.sql
```

3. Configure o ambiente. O `JWT_SECRET` precisa ter 32 caracteres ou mais:
```bash
cp .env.example .env
openssl rand -base64 48    # cole o resultado em JWT_SECRET no .env
```

4. Rode a API (http://localhost:8080):
```bash
set -a && source .env && set +a
go run ./cmd/api
```

Para rodar os testes: `go test ./...`

## Variáveis de ambiente
| Variável | Obrigatória | Descrição |
|---|---|---|
| `DATABASE_URL` | sim | Conexão com o PostgreSQL |
| `JWT_SECRET` | sim | Segredo do token, com 32 caracteres ou mais. A API não inicia com o valor de exemplo |
| `PORT` | não | Porta da API (padrão `8080`; o Render define sozinho) |
| `CORS_ORIGIN` | não | Origem(ns) do front liberadas no CORS, separadas por vírgula (padrão `http://localhost:5173`) |

## Rotas
| Rota | Login | O que faz |
|---|---|---|
| `GET /api/saude` | não | Health check (`{"status":"ok"}`) |
| `POST /api/usuarios` | não | Cadastra um usuário |
| `POST /api/login` | não | Autentica e devolve o token |
| `GET /api/produtos` | não | Lista os produtos ativos (aceita `?busca=`) |
| `GET /api/produtos/{id}` | não | Detalhe de um produto |
| `POST /api/pedidos` | sim | Registra um pedido |
| `GET /api/pedidos/{id}` | sim | Consulta um pedido (só do dono) |

As rotas com login pedem o header `Authorization: Bearer <token>`. Requests, respostas e códigos de erro estão em [`docs/API.md`](docs/API.md).

## Deploy (Render)
1. Crie um PostgreSQL na nuvem (Render ou Neon) e copie a URL de conexão. Ela precisa terminar com `?sslmode=require`.
2. Aplique o schema e o seed no banco da nuvem:
```bash
psql "<URL do banco>" -f db/schema.sql
psql "<URL do banco>" -f db/seed.sql
```
3. No Render, crie um **Web Service** ligado ao repositório (runtime Go) com:
   - Build Command: `go build -o app ./cmd/api`
   - Start Command: `./app`
   - Health Check Path: `/api/saude`
4. Defina as variáveis de ambiente: `DATABASE_URL`, `JWT_SECRET` (32 caracteres ou mais) e `CORS_ORIGIN` (endereço do front na Vercel, por exemplo `https://seu-app.vercel.app`). A `PORT` o Render já define.

No plano grátis, o serviço dorme após um tempo sem uso, então a primeira requisição depois disso demora um pouco.

## Estrutura
```
cmd/api/             ponto de entrada (servidor HTTP)
internal/config/     leitura das variáveis de ambiente
internal/db/         conexão e consultas ao PostgreSQL
internal/auth/       bcrypt, JWT e middleware de login
internal/handlers/   uma rota por função (usuários, produtos, pedidos)
internal/httpx/      JSON, erros e CORS
internal/models/     tipos das respostas
db/                  schema.sql e seed.sql
docs/                contrato da API
```

## Observações
- As senhas são guardadas com bcrypt, nunca em texto puro.
- O token JWT vale por 24 horas.
- O total do pedido é calculado no servidor, com os preços do banco. Valores enviados pelo cliente são ignorados.
- Pedido e itens são gravados em uma única transação: se algo falhar, nada é salvo.
- `GET /api/pedidos/{id}` só responde para quem fez o pedido. Para os outros usuários, o pedido aparece como não encontrado (404).
- As imagens dos cupcakes são servidas pelo front. O banco só guarda o caminho em `imagem_url` (ex.: `/img/morango.png`).
