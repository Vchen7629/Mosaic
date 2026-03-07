from typing import Annotated
from fastapi import status
from fastapi import HTTPException
from pymongo.errors import PyMongoError
from pymongo.asynchronous.database import AsyncDatabase
from fastapi import Depends
from fastapi import APIRouter
from ..db.conn import get_db
from ..db.conversation import get_all_conversations
from ..db.models import ConversationRecord
from ..db.conversation import add_conversation
from ..routes.models import AddConversationRequest
from ..routes.models import FetchAllConversationRequest
from ..Conversation.gemnisummerize import SummeriseWithGemina
SESSION_COOKIE_NAME = "session_token"


router = APIRouter(prefix="/conversation", tags=["auth"])


@router.post("/summerise")
async def signup(request: str):
    return SummeriseWithGemina(request)


@router.post("/save", status_code=status.HTTP_204_NO_CONTENT)
async def save_conversation(
    convo_details: AddConversationRequest,
    db: Annotated[AsyncDatabase, Depends(get_db)]
) -> None:
    try:
        await add_conversation(
            db, 
            convo_details.patient_id,
            convo_details.name,
            convo_details.timestamp,
            convo_details.conversation_summary,
            convo_details.topics
        ) 
    except PyMongoError as e:
        raise HTTPException(status_code=500, detail=f"Database error: {e}")


@router.post("/fetch_all")
async def fetch_all_conversations(
    request: FetchAllConversationRequest,
    db: Annotated[AsyncDatabase, Depends(get_db)]
) -> list[ConversationRecord]:
    try:
        return await get_all_conversations(db, request.patient_id, request.name)
    except PyMongoError as e:
        raise HTTPException(status_code=500, detail=f"Database error: {e}")

