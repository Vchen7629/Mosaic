
from pydantic import BaseModel

class AddConversationRequest(BaseModel):
    patient_id: str 
    name: str
    timestamp: str
    conversation_summary: str
    topics: list[str]

class FetchAllConversationRequest(BaseModel):
    patient_id: str
    name: str

class SummarizeConvoRequest(BaseModel):
    conversation: str

class GenerateBriefingRequest(BaseModel):
    patient_id: str
    name: str
    
class fetch_briefing_Request(BaseModel):
    patient_id: str
    name: str