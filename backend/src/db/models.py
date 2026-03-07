from typing import Optional
from pydantic import BaseModel


# A conversation summarized by gemini
class ConversationRecord(BaseModel):
    timestamp: str
    summary: str
    topics: list[str]
    last_convo_briefing: Optional[str] = None


# this is for each different person the patient talked to
class InteractedUser(BaseModel):
    name: str
    face_embedding: Optional[list[float]] = None  # numpy ndarray
    relationship: str
    conversations: list[ConversationRecord]


class PatientData(BaseModel):
    user_id: str
    face_embedding: Optional[list[float]] = None  # numpy ndarray
    interacted_users: Optional[list[InteractedUser]] = None
