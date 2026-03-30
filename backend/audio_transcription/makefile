test_all: integration unit

unit:
	uv run pytest test/unit/

integration:
	uv run pytest test/integration/

coverage:
	uv run pytest --cov=src --cov-report= test/unit/
	uv run pytest --cov=src --cov-report=xml:coverage.xml --cov-append test/integration/