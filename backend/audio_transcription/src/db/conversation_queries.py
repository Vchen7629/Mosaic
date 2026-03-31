from ..core.logging import logger
from ..core.metrics import DBWritesTotal
from .retry import retry_with_backoff
import psycopg


@retry_with_backoff(max_retries=3, initial_delay=1.0)
def save_conversation(
    conn: psycopg.Connection, profile_id: int, convo_text: str, visitor_ids: list[int]
) -> None:
    """
    Save a transcript of the conversation to the database
    for a profile's visitor

    Args:
        conn: the psycopg db connection
        profile_id: profile_id to identify which profile to
        save for
        convo_text: entire conversation in text
        visitor_id: the id of the visitor to save for
    """
    if not profile_id or profile_id <= 0:
        raise ValueError("invalid profile_id provided")
    if not convo_text:
        raise ValueError("invalid convo text provided")
    if not visitor_ids or not all(
        isinstance(vis_id, int) and vis_id > 0 for vis_id in visitor_ids
    ):
        raise ValueError("invalid visitor_ids provided")

    query = """
        INSERT INTO conversation_records (
            profile_id, visitor_id, created_at, convo_text
        ) VALUES (
            %(profile_id)s, %(visitor_id)s, NOW(), %(convo_text)s
        )"""

    try:
        for visitor_id in visitor_ids:
            DBWritesTotal.labels(operation="save_conversation").inc()

            with conn.cursor() as cursor:
                cursor.execute(
                    query,
                    {
                        "profile_id": profile_id,
                        "visitor_id": visitor_id,
                        "convo_text": convo_text,
                    },
                )
            logger.debug(
                "added conversation for visitor",
                visitor_id=visitor_id,
                profile_id=profile_id,
            )
        conn.commit()
    except Exception as e:
        logger.error(
            "failed to add conversation for visitor",
            visitor_ids=visitor_ids,
            profile_id=profile_id,
            err=str(e),
        )
        conn.rollback()
        raise
