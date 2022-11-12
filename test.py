# This example requires the 'message_content' intent.

import discord


class MyClient(discord.Client):
    async def on_ready(self):
        print(f'Logged on as {self.user}!')

    async def on_message(self, message):
        print(f'Message from {message.author}: {message.content}')


intents = discord.Intents.default()
intents.message_content = True

client = MyClient(intents=intents)
client.run(
    'MTA0MDgzMDUwODI5NDIwOTYwNg.GPb_80.hAbFodbd6fss1qiK4hOTV4tDztkHxYlOrS6CAc')
