from typing import Optional
from ..core.logging import logger
from ..core.metrics import ErrorsTotal
import psycopg


def fetch_profile_id_from_session(
    conn: psycopg.Connection, session_token: str
) -> Optional[int]:
    """
    Fetch the profile_id associated with a session_token from the sessions table.
    Returns None if not found.
    """
    try:
        with conn.cursor() as cursor:
            cursor.execute(
                "SELECT profile_id FROM sessions WHERE session_token = %s",
                (session_token,),
            )
            row = cursor.fetchone()
            return row[0] if row else None
    except Exception as e:
        ErrorsTotal.labels("fetch_profile_id_from_session").inc()
        logger.error(
            "failed to fetch profile ID from session",
            session_token=session_token,
            err=str(e),
        )
        conn.rollback()
        raise
