package com.hotzfellas.hackathonfall

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.os.Bundle
import android.view.inputmethod.InputMethodManager
import android.widget.Button
import android.widget.EditText
import android.widget.ImageView
import androidx.appcompat.app.AppCompatActivity
import com.chaquo.python.Python
import com.chaquo.python.android.AndroidPlatform
import java.util.*

class MainActivity : AppCompatActivity() {
    // Array of 4 bitmaps
    private var bitmaps = arrayOfNulls<Bitmap>(4)
    private var imageIndex = 0

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        if (!Python.isStarted()) {
            Python.start(AndroidPlatform(this))
        }
        val py = Python.getInstance()

        val input = findViewById<EditText>(R.id.Edit_Text_Prompt)
        val button = findViewById<Button>(R.id.Button_Generate)

        button.setOnClickListener {
            // Hide the keyboard
            val imm = getSystemService(Context.INPUT_METHOD_SERVICE) as InputMethodManager
            imm.hideSoftInputFromWindow(it.windowToken, 0)

            // Load the Dalle module
            val dallePrompt = py.getModule("DallePrompt")

            val weatherGetter = dallePrompt.callAttr("Weather")
            val temp = weatherGetter["temp"].toString()
            val humidity = weatherGetter["humidity"].toString()
            val speed = weatherGetter["speed"].toString()
            val weather = weatherGetter["weather"].toString()

            // Get the time of day
            val calendar = Calendar.getInstance()
            val hour = calendar.get(Calendar.HOUR_OF_DAY)
            // Morning, afternoon, evening, night
            val timeOfDay = when (hour) {
                in 0..5 -> "night"
                in 6..11 -> "morning"
                in 12..17 -> "afternoon"
                in 18..23 -> "evening"
                else -> "morning"
            }

            // Read the prompt
            val prompt =
                "${input.text}, $weather with a temperature of $temp degrees and a wind speed of $speed miles per hour in the $timeOfDay with a humidity of $humidity percent."

            // Generate the images
            val dp = dallePrompt.callAttr("DallePrompt", prompt)
            // Returns a list of URLs
            val pyUrls = dp["image_paths"]!!.asList()
            var urls: Vector<String> = Vector<String>()
            for (rc in pyUrls)
                urls.add(rc.toString())

            // Get the image from the internet
            val url = urls[0]
            val image = findViewById<ImageView>(R.id.Photo_Image_View)
            Thread {
                try {
                    val imageStream = java.net.URL(url).openStream()

                    imageStream?.let {
                        bitmaps[0] = BitmapFactory.decodeStream(it)

                        runOnUiThread(Runnable {
                            image.setImageBitmap(bitmaps[0])
                            image.visibility = ImageView.VISIBLE

                            // Make the button invisible
                            button.visibility = Button.INVISIBLE

                            // Make the input invisible
                            input.visibility = EditText.INVISIBLE
                        })

                    }
                } catch (e: Exception) {
                    e.printStackTrace()
                }
            }.start()

            // Load the rest of the images
            for (i in 1..3) {
                val url = urls[i]
                Thread {
                    try {
                        val imageStream = java.net.URL(url).openStream()

                        imageStream?.let {
                            bitmaps[i] = BitmapFactory.decodeStream(imageStream)
                        }
                    } catch (e: Exception) {
                        e.printStackTrace()
                    }
                }.start()
            }

            // Add on click listener to the image
            image.setOnClickListener {
                // Cycle through the images
                imageIndex = (imageIndex + 1) % 4
                image.setImageBitmap(bitmaps[imageIndex])
            }
        }
    }
}