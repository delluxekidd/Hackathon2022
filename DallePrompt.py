from dalle2 import Dalle2
import requests

# Weather Class
# Stores: Temperature, Humidity, Wind Speed, Weather Description


class Weather:
    def __init__(self):
        self._weather_response = requests.get(
            'http://api.openweathermap.org/data/2.5/forecast?lat=36.082157&lon=-94.171852&id=524901&id=524901&appid=bd0d4417e907059d5e1f3731e52fca94')

        for i in range(0, 40):
            if self._weather_response.json()['list'][i]['dt_txt'][11:13] == '15':
                self.temp = self._weather_response.json()[
                    'list'][i]['main']['temp']
                self.humidity = self._weather_response.json()[
                    'list'][i]['main']['humidity']
                self.wind = self._weather_response.json()[
                    'list'][i]['wind']['speed']
                self.weather = self._weather_response.json()[
                    'list'][i]['weather'][0]['description']
                break

    def __str__(self):
        return "Temperature: " + str(self.temp) + " Humidity: " + str(self.humidity) + " Wind Speed: " + str(self.wind) + \
            " Weather Description: " + str(self.weather)


class DallePrompt(Dalle2):
    def __init__(self, prompt):
        # this is the bearer key taken from dalle2
        self._dalle = Dalle2("sess-hSFU7aZjRvetjOoxT4O9zcx2s4r1WWWouiGCTRbx")

        # this is the prompt to pass to the dalle2 model
        generations = self._dalle.generate(prompt)

        # Loop through each element in the generations list and get the image path from the generation and add it to a list
        self.image_paths = []
        for generation in generations:
            self.image_paths.append(generation["generation"]["image_path"])
