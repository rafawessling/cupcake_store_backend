# Cupcake Store — Contrato da API

Base: `http://localhost:8080`. CORS liberado somente para as origens em `CORS_ORIGIN` (uma ou várias separadas por vírgula; padrão `http://localhost:5173`) (métodos `GET, POST, OPTIONS`; headers `Authorization, Content-Type`).
Todas as requisições/respostas são JSON (`Content-Type: application/json`). Corpo limitado a 1 MB.

## Convenções
- **Dinheiro**: número JSON com 2 casas decimais (`13.50`, `40.00`). Nunca enviar valores monetários: o back calcula tudo.
- **Autenticação**: header `Authorization: Bearer <token>` (JWT HS256, válido por 24h). O `token` vem de `POST /api/login`.
- **Erro** (qualquer status ≥ 400 devolvido pela API): `{"erro": "mensagem em português"}`
- **Datas**: `criado_em` em RFC 3339 UTC (`2026-11-16T14:30:00Z`).
- **401** em rota autenticada (header ausente, formato errado, assinatura inválida ou token expirado): `{"erro":"Token de acesso ausente ou inválido"}`.
- Rota inexistente: `404 {"erro":"Rota não encontrada"}`. `{id}` não numérico/≤0: `400 {"erro":"ID inválido"}`.
- Corpo JSON malformado: `400 {"erro":"Corpo da requisição inválido"}`; acima de 1 MB: `413`.
- Erro interno: `500 {"erro":"<mensagem genérica>"}` (detalhes só no log do servidor).

## GET /api/saude — health check (público)
- `200` → `{"status":"ok"}` (não consulta o banco; serve para o health check do Render)

## POST /api/usuarios — cadastrar (público)
Request:
```json
{"nome":"Ana Souza","email":"ana@mail.com","senha":"123456","confirmacao_senha":"123456"}
```
- `201` → `{"id":1,"nome":"Ana Souza","email":"ana@mail.com"}` (e-mail normalizado em minúsculas; senha gravada com bcrypt)
- `400` → mensagens: `Nome é obrigatório` · `Nome deve ter no máximo 120 caracteres` · `E-mail inválido` · `Senha deve ter ao menos 6 caracteres` · `Senha deve ter no máximo 72 caracteres` · `As senhas não conferem`
- `409` → `{"erro":"E-mail já cadastrado"}` (sem diferenciar maiúsculas)

## POST /api/login — autenticar (público)
Request: `{"email":"ana@mail.com","senha":"123456"}`
- `200` → `{"token":"<jwt>","usuario":{"id":1,"nome":"Ana Souza","email":"ana@mail.com"}}`
- `400` → `Informe e-mail e senha`
- `401` → `{"erro":"E-mail ou senha inválidos"}` (mesma mensagem para e-mail inexistente ou senha errada)

## GET /api/produtos — listar ativos (público)
Query opcional: `?busca=morango` (parte do nome, sem diferenciar maiúsculas).
- `200` → array (vazio `[]` se nada encontrado), ordenado por `id`:
```json
[{"id":1,"nome":"Morango Sonhador","descricao":"...","preco":13.00,"imagem_url":"/img/morango.png"}]
```

## GET /api/produtos/{id} — detalhe (público)
- `200` → `{"id":2,"nome":"Beijo de Berries","descricao":"...","preco":13.50,"imagem_url":"/img/berries.png"}`
- `404` → `{"erro":"Produto não encontrado"}` (inexistente **ou inativo**)
- `400` → `ID inválido`

## POST /api/pedidos — registrar pedido (autenticado)
Request (apenas `produto_id` e `quantidade`; qualquer preço/total enviado é ignorado):
```json
{"itens":[{"produto_id":1,"quantidade":2},{"produto_id":2,"quantidade":1}]}
```
Regras: 1–100 itens; `quantidade` de 1 a 99; `produto_id` sem repetição; produto deve existir e estar ativo. Preços são lidos do banco no momento do pedido. Pedido + itens gravados em uma única transação.

- `201`:
```json
{
  "id": 7,
  "status": "confirmado",
  "valor_total": 39.50,
  "criado_em": "2026-11-16T14:30:00Z",
  "itens": [
    {"produto_id":1,"nome":"Morango Sonhador","quantidade":2,"preco_unitario":13.00,"subtotal":26.00},
    {"produto_id":2,"nome":"Beijo de Berries","quantidade":1,"preco_unitario":13.50,"subtotal":13.50}
  ]
}
```
- `400` → `O pedido precisa ter ao menos um item` · `Pedido com itens demais` · `Produto inválido` · `Quantidade deve estar entre 1 e 99` · `Produto repetido no pedido` · `Produto {id} não existe ou está indisponível` (nada é gravado)
- `401` → token ausente/inválido/expirado
- `500` → `{"erro":"Não foi possível concluir o pedido"}` (rollback completo)

## GET /api/pedidos/{id} — consultar pedido (autenticado, somente dono)
- `200` → mesmo formato do `201` acima
- `404` → `{"erro":"Pedido não encontrado"}` (inexistente **ou de outro usuário**)
- `400` → `ID inválido` · `401` → token ausente/inválido/expirado

## Execução
Variáveis: `DATABASE_URL` (obrigatória), `JWT_SECRET` (obrigatória, ≥32 caracteres, não pode ser o valor de exemplo; gere com `openssl rand -base64 48`), `PORT` (padrão `8080`), `CORS_ORIGIN` (origens separadas por vírgula, sem barra final; padrão `http://localhost:5173`). Ver `.env.example` e o `README.md`.
