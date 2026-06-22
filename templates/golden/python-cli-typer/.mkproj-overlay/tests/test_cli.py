from typer.testing import CliRunner

from app.main import app


def test_hello_command_walks() -> None:
    runner = CliRunner()

    result = runner.invoke(app, ["hello", "Peter"])

    assert result.exit_code == 0
    assert "hello, Peter!" in result.stdout
