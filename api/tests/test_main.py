from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def test_health():
    assert client.get("/health").json() == {"status": "ok"}


def test_validate_accepts_select():
    r = client.post("/api/sql/validate", json={"text": "select 1;"})
    assert r.status_code == 200
    assert r.json()["sql"] == "SELECT 1"


def test_validate_rejects_write():
    r = client.post("/api/sql/validate", json={"text": "delete from t"})
    assert r.status_code == 400
    assert "SELECT" in r.json()["detail"] or "not allowed" in r.json()["detail"]
