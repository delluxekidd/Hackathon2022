from dalle2 import Dalle2
import json

dalle = Dalle2("sess-DQZKfol1LYPw7CufffheuOqHZlFy27VQC7I15qW2") # this is the bearer key taken from dalle2

generations = dalle.generate("a dog holding a gun, mafia, realistic, real life, 4k") # this is the prompt to pass to the dalle2 model

# Loop through each element in the generations list and get the image path from the generation and add it to a list
image_paths = []
for generation in generations:
    image_paths.append(generation["generation"]["image_path"])

print(image_paths)
