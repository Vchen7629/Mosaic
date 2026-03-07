from typing import Optional
from pymongo.asynchronous.database import AsyncDatabase
from ..db.models import PatientData

async def testing_insert_embeddings(db: AsyncDatabase):
    collection = db["embeddings"]

    await collection.insert_one({"user_id": "abc", "embedding": ["hey bro"]})

async def get_patient_doc(db: AsyncDatabase, patient_id: str) -> Optional[PatientData]:
    collection = db["patients"]
    doc = await collection.find_one({"user_id": patient_id})

    if doc:
        doc["_id"] = str(doc["_id"])
        
    return doc

async def add_interacted_user(
    db: AsyncDatabase, 
    patient_id: str, 
    name: str,
    embedding: list[float]
) -> None:
    collection = db["patients"]

    await collection.update_one(
        {"user_id": patient_id},
        {"$push": {"interacted_users": {
            "name": name,
            "face_embedding": embedding,
            "relationship": None,
            "conversations": []
        }}},
        upsert=True
    )