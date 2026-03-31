from valkey import Valkey
from typing import Optional
from ..core.metrics import ErrorsTotal


def fetch_profile_id_from_cache(cache: Valkey, session_token: str) -> Optional[int]:
    """
    Fetch the profile_id associated with a session_token from valkey cache.
    Returns None if the key is not found or on error.
    """
    key = f"session:{session_token}"
    value = cache.get(key)
    if value is None:
        return None
    try:
        return int(value)  # type: ignore[arg-type]
    except (ValueError, TypeError):
        ErrorsTotal.labels("fetch_profile_id_from_cache").inc()
        return None
