from psycopg_pool import ConnectionPool
from ..core.settings import settings

def create_connection_pool() -> ConnectionPool:
    """
    Create a PG connection pool using env variables and 
    verifies its available

    Returns:
        ConnectionPool instance

    Raises:
        Exception and closes db_pool if db isnt reachable
    """
    
    conninfo = (
        f"host={settings.DB_HOST} "
        f"port={settings.DB_PORT} "
        f"dbname={settings.DB_NAME} "
        f"user={settings.DB_USER} "
        f"password={settings.DB_PASS} "
    )
    
    db_pool = ConnectionPool(
        conninfo=conninfo,
        min_size=settings.POOL_MIN_SIZE,
        max_size=settings.POOL_MAX_SIZE,
        open=True,
        kwargs={
            "keepalives": settings.KEEP_ALIVES_COUNT, 
            "keepalives_idle": settings.KEEP_ALIVES_IDLE_S,
            "keepalives_interval": settings.KEEP_ALIVES_RETRY_INTERVAL_S,
            "keepalives_count": settings.KEEP_ALIVES_RETRY_COUNT
        },
    )

    try:
        with db_pool.connection() as conn:
            with conn.cursor() as cursor:
                cursor.execute("SELECT 1")
                cursor.fetchone()
    except Exception as e:
        db_pool.close()
        raise

    return db_pool