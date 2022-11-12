from dalle2 import Dalle2
import pygame
import urllib.request
import sys
import requests

# this is the bearer key taken from dalle2
dalle = Dalle2("sess-hSFU7aZjRvetjOoxT4O9zcx2s4r1WWWouiGCTRbx")

weather = requests.get(
    'http://api.openweathermap.org/data/2.5/forecast?lat=36.082157&lon=-94.171852&id=524901&id=524901&appid=bd0d4417e907059d5e1f3731e52fca94')

# Weather Class
# Stores: Temperature, Humidity, Wind Speed, Weather Description


class Weather:
    def __init__(self, temp, humidity, wind, weather):
        self.temp = temp
        self.humidity = humidity
        self.wind = wind
        self.weather = weather

    def __str__(self):
        return "Temperature: " + str(self.temp) + " Humidity: " + str(self.humidity) + " Wind Speed: " + str(self.wind) + \
            " Weather Description: " + str(self.weather)


# Weather API
# Returns: Weather Object
# Parses JSON data from OpenWeatherMap API
for i in range(0, 40):
    if weather.json()['list'][i]['dt_txt'][11:13] == '15':
        temp = weather.json()['list'][i]['main']['temp']
        humidity = weather.json()['list'][i]['main']['humidity']
        wind = weather.json()['list'][i]['wind']['speed']
        weather = weather.json()['list'][i]['weather'][0]['description']
        break

# Create Weather Object
weather = Weather(temp, humidity, wind, weather)

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

# this is the prompt to pass to the dalle2 model
generations = dalle.generate(prompt)

# Loop through each element in the generations list and get the image path from the generation and add it to a list
image_paths = []
for generation in generations:
    image_paths.append(generation["generation"]["image_path"])

# Loop through each file path in the image_paths list and print the out to files
i = 0
for image_path in image_paths:
    # Save image to file
    urllib.request.urlretrieve(image_path, "image" + str(i) + ".png")
    i += 1

# Pygame func to display the image


def display_image(image_path):
    # Open the image in a pygame window
    pygame.init()

    # Get screen size
    screen = pygame.display.set_mode((0, 0), pygame.FULLSCREEN)
    screen_width = screen.get_width()
    screen_height = screen.get_height()

    # Load image
    image = pygame.image.load(image_path)

    # Display image in center of screen
    image_width = image.get_width()
    image_height = image.get_height()
    screen.blit(image, ((screen_width - image_width) /
                2, (screen_height - image_height) / 2))

    # Display image
    pygame.display.flip()

    # Wait for user press space to close
    # Open next image in new window
    i = 0
    while True:
        # sleep for 10ms
        pygame.time.delay(100)
        for event in pygame.event.get():
            if event.type == pygame.KEYDOWN:
                if event.key == pygame.K_SPACE:
                    # Load image
                    i += 1
                    image = pygame.image.load("image" + str(i) + ".png")

                    # Display image in center of screen
                    image_width = image.get_width()
                    image_height = image.get_height()
                    screen.blit(image, ((screen_width - image_width) /
                                2, (screen_height - image_height) / 2))

                    # Display image
                    pygame.display.flip()
                if event.key == pygame.K_ESCAPE:
                    pygame.quit()
                    sys.exit()


display_image("image0.png")
