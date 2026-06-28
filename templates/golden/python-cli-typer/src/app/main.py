import typer

app = typer.Typer(no_args_is_help=True)


@app.command()
def hello(name: str = "world") -> None:
    print(f"hello, {name}!")


def main() -> None:
    app()


if __name__ == "__main__":
    main()
