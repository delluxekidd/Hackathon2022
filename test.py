from DallePrompt import DallePrompt, Weather

weather = Weather()
print(str(weather))


prompt = ""
choice = ""

# Ask user for choice of auto or manual
# Auto will generate a random prompt based on the weather and other factors
# Manual will ask the user for a prompt
while choice != "auto" and choice != "manual":
    choice = input(
        "Would you like to generate a random prompt or enter your own? (auto/manual): ")

if choice == "auto":
    # Generate a random prompt from weather data
    print("Generating random prompt...")
    prompt = "A picture of a " + str(weather.weather) + " with a temperature of " + str(
        weather.temp) + " degrees and a wind speed of " + str(weather.wind) + " miles per hour" + ", artistic, 4k"
    print("Prompt: " + prompt)
    # wait for user to press enter
    input("Press enter to generate image...")

if choice == "manual":
    # Ask user for prompt
    prompt = input("Enter your prompt: ")

# Ask user for prompt text
while prompt == "":
    prompt = input("Enter prompt text: ")


dalle_prompt = DallePrompt(prompt)

print(str(dalle_prompt.image_paths))
