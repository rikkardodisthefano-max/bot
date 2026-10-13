# RikksLabels — Bot-Pro

Bot de clientes e atendimento da **RikksLabels**. Ele reúne gerenciamento de clientes, pagamentos e atendimento por IA com memória por ticket, de forma simples e automatizada.

**Servidor:** https://discord.gg/cvRMmFJnbv

## Configuração administrativa

Defina no ambiente do deploy:

- `BOT_ADMIN_PASSWORD` — senha administrativa secreta.
- `DISCORD_BOT_TOKEN` — token do bot.
- `GEMINI_API_KEY` — chave da Gemini.

Use `/painel-admin` no Discord para abrir o painel protegido. O acesso exige a permissão **Administrador** e a senha. Após 3 tentativas incorretas, o usuário fica bloqueado permanentemente.

## Comandos administrativos

- `/atividade` — altera a atividade do bot.
- `/editar-ticket` — personaliza título, mensagem e rodapé dos novos tickets.
- `/limites-ia` — configura RPM e RPD por servidor.
- `/painel-admin` — abre o painel protegido.

Placeholders aceitos na mensagem do ticket: `{usuario}` e `{categoria}`.
