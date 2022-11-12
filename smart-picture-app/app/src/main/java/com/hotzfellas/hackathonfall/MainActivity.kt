package com.hotzfellas.hackathonfall

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Typeface
import android.os.Bundle
import android.view.KeyEvent
import android.view.MotionEvent
import android.view.inputmethod.InputMethodManager
import android.widget.*
import androidx.appcompat.app.AppCompatActivity
import androidx.constraintlayout.widget.ConstraintLayout
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

            // Change temperature to warm, cold, or neutral
            val tempV = when {
                temp.toFloat() > 303 -> "warm"
                temp.toFloat() < 288 -> "cold"
                else -> "neutral"
            }

            // Generate a random number between 0 and 4
            val rand = Random()
            val randNum = rand.nextInt(5)

            // Set randomMod to certain strings based on the random number
            val randomMod = when (randNum) {
                0 -> "hyperrealistic"
                1 -> "water color"
                2 -> "abstract"
                3 -> "cartoon"
                4 -> "wallpaper"
                else -> ""
            }

            // Read the prompt
            val prompt =
                "${input.text}, $weather day, $tempV, $timeOfDay, $randomMod"

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
                            button.visibility = Button.GONE

                            // Make the input invisible
                            input.visibility = EditText.GONE

                            // Make the other button visible
                            val button2 = findViewById<Button>(R.id.Move_Button)
                            button2.visibility = Button.VISIBLE
                            // Make it transparent
                            button2.alpha = 0.0f
                            button2.isSelected = true

                            // Set the value of the clock
                            val textClock = TextClock(this)
                            textClock.format12Hour = "h:mm a"
                            textClock.format24Hour = "H:mm"
                            textClock.textSize = 100F
                            textClock.setTypeface(null, Typeface.BOLD)
                            textClock.setTextColor(-0x777778)

                            // Add the clock to the layout
                            val clock_layout = findViewById<LinearLayout>(R.id.Clock_View)
                            clock_layout.addView(textClock)

                            // Create small text views
                            val tempView = TextView(this)
                            // Convert from Kelvin to Fahrenheit
                            tempView.text = "Temperature: ${((temp.toFloat() - 273.15) * 9 / 5 + 32).toInt()}°F"
                            tempView.textSize = 40F
                            textClock.setTypeface(null, Typeface.BOLD)
                            tempView.setTextColor(-0x777778)

                            // Add the temperature to the layout
                            val info_layout = findViewById<LinearLayout>(R.id.Info_View)
                            info_layout.addView(tempView)

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
            val button2 = findViewById<Button>(R.id.Move_Button)
            button2.setOnClickListener {
                // Cycle through the images
                imageIndex = (imageIndex + 1) % 4
                image.setImageBitmap(bitmaps[imageIndex])
            }
        }
    }
}