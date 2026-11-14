import os

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from app.sqlguard import MAX_SQL_CHARS, GuardError, validate

ALLOWED_SCHEMAS = {
    s.strip().lower() for s in os.getenv("BI_ALLOWED_SCHEMAS", "public,ml").split(",") if s.strip()
}

app = FastAPI(title="hst-bi-api", version="0.1.0")


class SqlRequest(BaseModel):
    text: str = Field(max_length=MAX_SQL_CHARS)


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/sql/validate")
def validate_sql(body: SqlRequest) -> dict[str, str]:
    """Dry check only. Query execution arrives in P4 with the database pool."""
    try:
        return {"sql": validate(body.text, ALLOWED_SCHEMAS)}
    except GuardError as err:
        raise HTTPException(status_code=400, detail=str(err)) from err
