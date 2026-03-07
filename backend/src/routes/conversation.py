from fastapi import APIRouter, Depends, HTTPException, status, Response, Request
from ..Conversation.gemnisummerize import SummeriseWithGemina
SESSION_COOKIE_NAME = "session_token"


router = APIRouter(prefix="/conversation", tags=["auth"])


@router.get("/summerise")
async def signup(request: str):
    return SummeriseWithGemina(request)