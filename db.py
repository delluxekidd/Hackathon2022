# Using flask to make an api
# import necessary libraries and functions
from flask import Flask, jsonify, request
from DallePrompt import DallePrompt, Weather
from threading import Thread
import time

# creating a Flask app
app = Flask(__name__)

image_paths = []

def getImages(prompt):
    weather_getter = Weather()
    temp = weather_getter.temp
    speed = weather_getter.wind
    humidity = weather_getter.humidity
    weather = weather_getter.weather

    # Get the time of day
    currentTime = int(time.strftime("%H"))
    timeOfDay = "night"
    if currentTime > 18:
        timeOfDay = "night"
    elif currentTime > 12:
        timeOfDay = "afternoon"
    elif currentTime > 6:
        timeOfDay = "morning"

    prompt += f', {weather} with a temperature of {temp} degrees and a wind speed of {speed} miles per hour in the {timeOfDay} with a humidity of {humidity} percent.'

    dalle_prompt = DallePrompt(prompt)
    global image_paths
    image_paths = dalle_prompt.image_paths

@app.route('/prompt', methods = ['GET', 'POST'])
def prompt():
    if(request.method == 'GET'):
        # Return code 200
        if(len(image_paths) > 0):
            return jsonify(image_paths), 200
        else:
            return jsonify("No images found"), 404

    if(request.method == 'POST'):
        image_paths.clear()
        data = request.get_json()

        # Get prompt from request
        prompt = data['prompt']

        # Start thread that will generate image
        Thread(target=getImages, args=(prompt,)).start()

        return jsonify({'message': 'Ok'}), 200


# driver function
if __name__ == '__main__':
    app.run(debug = True)