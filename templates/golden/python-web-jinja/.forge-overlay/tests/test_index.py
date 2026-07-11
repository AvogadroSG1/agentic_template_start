from fastapi.testclient import TestClient

from app.main import app


def test_index_page_renders_the_skeleton() -> None:
    client = TestClient(app)

    response = client.get("/")

    assert response.status_code == 200
    assert "API health" in response.text
    assert "/static/htmx.min.js" in response.text


def test_htmx_is_served_locally() -> None:
    client = TestClient(app)

    response = client.get("/static/htmx.min.js")

    assert response.status_code == 200
