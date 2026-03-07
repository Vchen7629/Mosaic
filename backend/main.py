
from fastapi import FastAPI
from src.routes.conversation import router

app = FastAPI()
app.include_router(router)

@app.get("/")
async def root():
    
    return {"message": "Hello World"}


