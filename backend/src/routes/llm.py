from ..db.conversation import save_briefing, get_all_conversations
from ..llm.summarize_transcript import generate_visitor_briefing, SummarizeConversation
from ..llm.parse_conversations import extract_summaries_timestamps
from ..db.conn import get_db
from .models import SummarizeConvoRequest, GenerateBriefingRequest
from fastapi import APIRouter, Depends, HTTPException, status
from pymongo.asynchronous.database import AsyncDatabase
from typing import Annotated

router = APIRouter(prefix="/llm", tags=["llm"])

@router.post("/summarize")
async def summarize_conversation(request: SummarizeConvoRequest):
    try:
        return await SummarizeConversation(request.conversation)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error summarizing: {e}"
        )

@router.post("/generate_briefing", status_code=status.HTTP_201_CREATED)
async def generate_briefing(
    request: GenerateBriefingRequest,
    db: Annotated[AsyncDatabase, Depends(get_db)],
):
    patient_id = request.patient_id
    name = request.name

    try:
        convos = await get_all_conversations(db, patient_id, name)

        timestamps, summaries = extract_summaries_timestamps(convos)

        briefing = await generate_visitor_briefing(summaries, timestamps)

        await save_briefing(db, patient_id, name, briefing)

    except Exception as e:
        raise