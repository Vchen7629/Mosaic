from dotenv import load_dotenv
import os
from google import genai

load_dotenv()
value = os.getenv("APIkey")
def SummeriseWithGemina(text):
    client = genai.Client(api_key = value)
    response = client.models.generate_content(
        model="gemini-2.5-flash",
        contents=f"Give a one to five sentance summery \n\n{text} \
        \n Create a list/bullet points of top 5 common topics talked about in this conversation. heres an example \
        *Textone \
        *Texttwo \
        *TextThree \
        *TextFour \
        *textFive"
    )

    return response