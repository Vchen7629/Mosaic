from pymongo import AsyncMongoClient
from pymongo.asynchronous.database import AsyncDatabase
from ..core.settings import settings

class Database:
    def __init__(self):
        self.client: AsyncMongoClient | None = None
        self._db: AsyncDatabase | None = None
        self._uri = settings.MONGODB_URI
        self._db_name = settings.DB_NAME

    async def connect(self):
        self.client = AsyncMongoClient(self._uri)
        self._db = self.client[self._db_name]

    async def disconnect(self):
        if self.client:
           await self.client.close()

    def get_db(self) -> AsyncDatabase:
        return self._db


db_instance = Database()


def get_db() -> AsyncDatabase:
    return db_instance.get_db()