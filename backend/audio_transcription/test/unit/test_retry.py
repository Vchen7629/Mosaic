from unittest.mock import patch, MagicMock, call
import pytest
import psycopg
from src.db.retry import retry_with_backoff


def _make_func(
    side_effects,
    max_retries=3,
    initial_delay=1.0,
    backoff_multiplier=2.0,
    max_delay=30.0,
):
    mock_fn = MagicMock(side_effect=side_effects)

    @retry_with_backoff(
        max_retries=max_retries,
        initial_delay=initial_delay,
        backoff_multiplier=backoff_multiplier,
        max_delay=max_delay,
    )
    def func():
        return mock_fn()

    return func, mock_fn


def test_retryable_error_logs_warnings_then_raises_and_logs_error_after_max_retries():
    func, _ = _make_func([psycopg.OperationalError("conn failed")] * 4)

    with (
        patch("src.db.retry.time.sleep"),
        patch("src.db.retry.logger") as mock_logger,
    ):
        with pytest.raises(psycopg.OperationalError):
            func()

    assert mock_logger.warning.call_count == 3
    mock_logger.error.assert_called_once_with(
        "failed after max retries", err="conn failed"
    )


def test_sleeps_with_exponential_backoff_capped_at_max_delay():
    func, _ = _make_func(
        [psycopg.OperationalError("err")] * 5,
        max_retries=4,
        initial_delay=10.0,
        max_delay=15.0,
        backoff_multiplier=2.0,
    )

    with (
        patch("src.db.retry.time.sleep") as mock_sleep,
        patch("src.db.retry.logger"),
    ):
        with pytest.raises(psycopg.OperationalError):
            func()

    assert mock_sleep.call_args_list == [call(10.0), call(15.0), call(15.0), call(15.0)]


def test_raises_immediately_without_sleeping_on_non_retryable_error():
    func, mock_fn = _make_func([ValueError("bad input")])

    with (
        patch("src.db.retry.time.sleep") as mock_sleep,
        patch("src.db.retry.logger"),
    ):
        with pytest.raises(ValueError):
            func()

    mock_sleep.assert_not_called()
    mock_fn.assert_called_once()


def test_succeeds_after_transient_failures():
    func, mock_fn = _make_func(
        [
            psycopg.OperationalError("transient"),
            psycopg.OperationalError("transient"),
            "ok",
        ]
    )

    with (
        patch("src.db.retry.time.sleep"),
        patch("src.db.retry.logger"),
    ):
        result = func()

    assert result == "ok"
    assert mock_fn.call_count == 3
