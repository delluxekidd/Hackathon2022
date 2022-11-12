# This example requires the 'message_content' intent.

import discord

intents = discord.Intents.default()
intents.message_content = True

client = discord.Client(intents=intents)


# bot is ready
@client.event
async def on_ready():
    try:
        print(client.user.name)
        print(client.user.id)
        print('Discord.py Version: {}'.format(discord.__version__))

        blah = discord.Permissions.use_application_commands
        print(blah)

        # Send a slash in the channel with ID 1040829566622638092
        channel = client.get_channel(1040829566622638092)
        await channel.send('/imagine being a bot')

    except Exception as e:
        print(e)


@client.event
async def on_message(message):
    # print message content in terminal
    print(message.content)


client.run(
    'MTA0MDgzMDUwODI5NDIwOTYwNg.G9YKRW.qeMo1bzEZSQZ9Pu6mmf9FQxASy36kIYBWmwoQc')
