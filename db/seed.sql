-- Cupcake Store — dados de exemplo (rodar após schema.sql, em banco recém-criado)
-- Uso: psql "$DATABASE_URL" -f db/seed.sql

INSERT INTO produtos (nome, descricao, preco, imagem_url) VALUES
('Morango Sonhador',
 'Massa macia de baunilha com recheio de geleia de morango e cobertura de chantilly rosa. Doce na medida certa.',
 13.00, '/img/morango.png'),
('Beijo de Berries',
 'Mistura de frutas vermelhas com um toque levemente azedinho e cobertura cremosa. Refrescante, vibrante e cheio de sabor em cada mordida.',
 13.50, '/img/berries.png'),
('Red Velvet Crush',
 'Clássico red velvet aveludado com cobertura de cream cheese e migalhas crocantes.',
 13.50, '/img/red_velvet.png'),
('Baunilha dos Sonhos',
 'Massa fofinha de baunilha com cobertura de buttercream sedoso e granulado colorido.',
 12.50, '/img/baunilha.png'),
('Chocolate Intenso',
 'Massa de chocolate meio amargo com recheio de brigadeiro e cobertura de ganache.',
 14.00, '/img/chocolate.png'),
('Lemon Cheesecake',
 'Massa cremosa e cítrica de limão com cobertura de cream cheese e farelo de biscoito amanteigado. Doce, azedinho na medida e irresistível.',
 13.00, '/img/limao.png');
