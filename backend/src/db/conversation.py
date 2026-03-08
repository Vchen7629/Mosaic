from ..db.models import ConversationRecord
from pymongo.asynchronous.database import AsyncDatabase
from pymongo.errors import PyMongoError

async def add_conversation(
    db: AsyncDatabase, 
    patient_id: str, 
    name: str,
    timestamp: str,
    conversation_summary: str,
    topics: list[str]
) -> None:
    collection = db["patients"]
    
    try:
        result = await collection.update_one(
            {"user_id": patient_id, "interacted_users.name": name},
            {"$push": {"interacted_users.$.conversations": {
                "timestamp": timestamp,
                "summary": conversation_summary,
                "topics": topics
            }}},
        )

        if result.matched_count == 0:
            await collection.update_one(
                {"user_id": patient_id},
                {"$push": {"interacted_users": {
                    "name": name,
                    "conversations": [{
                        "timestamp": timestamp,
                        "summary": conversation_summary,
                        "topics": topics
                    }]
                }}},
                upsert=True
            )
    except PyMongoError as e:
        print(f"Failed to update interacted_users: {e}")
        raise

async def save_briefing(
    db: AsyncDatabase,
    patient_id: str,
    name: str,
    briefing: str
) -> None:
    collection = db["patients"]

    try:
        result = await collection.update_one(
            {"user_id": patient_id, "interacted_users.name": name},
            {"$set": {"interacted_users.$.last_convo_briefing": briefing}}
        )
        if result.matched_count == 0:
            await collection.update_one(
                {"user_id": patient_id},
                {"$push": {"interacted_users": {"name": name, "last_convo_briefing": briefing}}},
                upsert=True
            )
    except PyMongoError as e:
        raise

async def get_all_conversations(
    db: AsyncDatabase, 
    patient_id: str, 
    name: str,
) -> list[ConversationRecord]:
    try:
        collection = db["patients"]
        patient = await collection.find_one({"user_id": patient_id})

        if not patient or not patient.get("interacted_users"):
            return []

        conversations = []
        name_convo = next((u for u in patient["interacted_users"] if u["name"] == name), None)
        if not name_convo:
            return []
        for convo in name_convo.get("conversations", []):
            conversations.append(ConversationRecord(**convo))

        return conversations
    except PyMongoError as e:
        raise


async def getbreifing(
    db: AsyncDatabase, 
    patient_id: str, 
    name: str,
):
    try:
        collection = db["patients"]
        patient = await collection.find_one({"user_id": patient_id})

        if not patient or not patient.get("interacted_users"):
            return []
        
        name_convo = next((u for u in patient["interacted_users"] if u["name"] == name), None)
        if not name_convo:
            return []
      
        test = name_convo.get{"last_convo_briefing"}

        return text
    except PyMongoError as e:
        raise