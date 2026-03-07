from typing import Optional
from pydantic import BaseModel


# A conversation summarized by gemini
class ConversationRecord(BaseModel):
    timestamp: str
    summary: str
    topics: list[str]


# this is for each different person the patient talked to
class InteractedUser(BaseModel):
    name: str
    face_embedding: Optional[list[float]] = None  # numpy ndarray
    conversations: list[ConversationRecord]
    last_convo_briefing: Optional[str] = None


class PatientData(BaseModel):
    user_id: str
    face_embedding: Optional[list[float]] = None  # numpy ndarray
    interacted_users: Optional[list[InteractedUser]] = None
