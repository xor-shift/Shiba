# Shiba

Some bot idk  
(golang)

## Stuff to run the bot:

- Create a DB: `sqlite3 botdb.sq3 -init ./bot/schema.sql`
- Build the bot: `go build -o ./shiba ./bot`
- Run the bot: `./shiba ./botdb.sq3`
- Pray that it runs
- Run migrate scripts if needed like: `sqlite3 botdb.sq3 < ./db/000_migrate_reactions.sql`
- Oh and you need to input information to for example the irc_configs table for the bot to do anything substantial
- Pray that it runs after configuring the bot
- ???
- Profit
