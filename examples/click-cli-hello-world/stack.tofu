# Define the Python Click CLI stack
resource "tofukit_stack" "python_click_cli" {
  name        = "python-click-cli"
  description = "Stack for building Python CLI applications with Click"

  depends_on = [
    tofukit_language.python,
    tofukit_tool.pip,
    tofukit_tool.venv,
    tofukit_framework.click,
    tofukit_methodology.idiomatic
  ]

  # Main application file
  scaffold {
    path = "app.py"
    content = <<-EOF
    #!/usr/bin/env python3
    """
    Hello World CLI application using Click framework.
    """
    import click
    from typing import Optional


    @click.command()
    @click.option(
        '--name',
        default='World',
        help='Name to greet',
        type=str
    )
    @click.option(
        '--count',
        default=1,
        help='Number of greetings',
        type=int
    )
    @click.option(
        '--shout',
        is_flag=True,
        help='Output in uppercase'
    )
    @click.version_option(version='0.1.0', prog_name='hello-cli')
    def hello(name: str, count: int, shout: bool) -> None:
        """Simple CLI that greets NAME COUNT times."""
        for _ in range(count):
            greeting = f'Hello, {name}!'
            if shout:
                greeting = greeting.upper()
            click.echo(greeting)


    if __name__ == '__main__':
        hello()
    EOF
  }

  # Requirements file
  scaffold {
    path = "requirements.txt"
    content = <<-EOF
    click==8.1.7
    EOF
  }

  # Development requirements
  scaffold {
    path = "requirements-dev.txt"
    content = <<-EOF
    click==8.1.7
    pytest==7.4.3
    black==23.11.0
    mypy==1.7.1
    ruff==0.1.6
    EOF
  }

  # Setup script
  scaffold {
    path = "setup.py"
    content = <<-EOF
    from setuptools import setup, find_packages

    with open("README.md", "r", encoding="utf-8") as fh:
        long_description = fh.read()

    setup(
        name="hello-cli",
        version="0.1.0",
        author="Your Name",
        author_email="your.email@example.com",
        description="A simple Hello World CLI using Click",
        long_description=long_description,
        long_description_content_type="text/markdown",
        url="https://github.com/yourusername/hello-cli",
        py_modules=["app"],
        classifiers=[
            "Programming Language :: Python :: 3",
            "License :: OSI Approved :: MIT License",
            "Operating System :: OS Independent",
        ],
        python_requires=">=3.8",
        install_requires=[
            "click>=8.0.0",
        ],
        entry_points={
            "console_scripts": [
                "hello-cli=app:hello",
            ],
        },
    )
    EOF
  }

  # README
  scaffold {
    path = "README.md"
    content = <<-EOF
    # Hello CLI

    A simple command-line application built with Python and Click framework.

    ## Installation

    1. Create a virtual environment:
       ```bash
       python3 -m venv .venv
       source .venv/bin/activate  # On Windows: .venv\Scripts\activate
       ```

    2. Install dependencies:
       ```bash
       pip install -r requirements.txt
       ```

    3. Install in development mode:
       ```bash
       pip install -e .
       ```

    ## Usage

    ```bash
    # Basic usage
    python app.py

    # With options
    python app.py --name "Alice" --count 3

    # Shout mode
    python app.py --name "Bob" --shout

    # Show version
    python app.py --version

    # Show help
    python app.py --help
    ```

    ## Development

    Install development dependencies:
    ```bash
    pip install -r requirements-dev.txt
    ```

    Run tests:
    ```bash
    pytest
    ```

    Format code:
    ```bash
    black app.py
    ```

    Check types:
    ```bash
    mypy app.py
    ```

    ## License

    MIT
    EOF
  }

  # Gitignore
  scaffold {
    path = ".gitignore"
    content = <<-EOF
    # Byte-compiled / optimized / DLL files
    __pycache__/
    *.py[cod]
    *$py.class

    # Virtual environment
    .venv/
    venv/
    ENV/
    env/

    # Distribution / packaging
    .Python
    build/
    develop-eggs/
    dist/
    downloads/
    eggs/
    .eggs/
    lib/
    lib64/
    parts/
    sdist/
    var/
    wheels/
    *.egg-info/
    .installed.cfg
    *.egg

    # Testing
    .pytest_cache/
    .coverage
    htmlcov/
    .tox/
    .mypy_cache/
    .ruff_cache/

    # IDE
    .idea/
    .vscode/
    *.swp
    *.swo
    *~

    # OS
    .DS_Store
    Thumbs.db
    EOF
  }

  # Simple test file
  scaffold {
    path = "test_app.py"
    content = <<-EOF
    """Tests for the hello CLI application."""
    import pytest
    from click.testing import CliRunner
    from app import hello


    def test_hello_default():
        """Test default behavior."""
        runner = CliRunner()
        result = runner.invoke(hello)
        assert result.exit_code == 0
        assert "Hello, World!" in result.output


    def test_hello_with_name():
        """Test with custom name."""
        runner = CliRunner()
        result = runner.invoke(hello, ["--name", "Alice"])
        assert result.exit_code == 0
        assert "Hello, Alice!" in result.output


    def test_hello_with_count():
        """Test multiple greetings."""
        runner = CliRunner()
        result = runner.invoke(hello, ["--count", "3"])
        assert result.exit_code == 0
        assert result.output.count("Hello, World!") == 3


    def test_hello_shout():
        """Test shout mode."""
        runner = CliRunner()
        result = runner.invoke(hello, ["--shout"])
        assert result.exit_code == 0
        assert "HELLO, WORLD!" in result.output


    def test_version():
        """Test version option."""
        runner = CliRunner()
        result = runner.invoke(hello, ["--version"])
        assert result.exit_code == 0
        assert "hello-cli, version 0.1.0" in result.output
    EOF
  }
}
