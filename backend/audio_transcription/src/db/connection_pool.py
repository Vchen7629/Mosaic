from psycopg_pool import ConnectionPool

def create_connection_pool(
    db_host: str,
    db_port: int,
    db_name: str,
    db_user: str,
    db_password: str,
    min_size: int = 1, 
    max_size: int = 10
) -> ConnectionPool:
    """
    Create a PG connection pool using env variables and 
    verifies its available

    Args:
        min_size: Minimum connections in pool
        max_size: Maximum connections in pool

    Returns:
        ConnectionPool instance

    Raises:
        Exception and closes db_pool if db isnt reachable
    """
    
    conninfo = (
        f"host={db_host} "
        f"port={db_port} "
        f"dbname={db_name} "
        f"user={db_user} "
        f"password={db_password} "
    )
    
    db_pool = ConnectionPool(conninfo=conninfo, min_size=min_size, max_size=max_size, open=True)

    try:
        with db_pool.connection() as conn:
            with conn.cursor() as cursor:
                cursor.execute("SELECT 1")
                cursor.fetchone()
    except Exception as e:
        db_pool.close()
        raise

    return db_pool