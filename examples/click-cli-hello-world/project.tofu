# ========================================
# CLICK CLI PROJECT DEFINITION
# ========================================
resource "tofukit_project" "click_cli" {
  name        = "python-click-cli"
  description = "Deterministic Python CLI application using Click framework"
  version     = "1.0.0"

  # Aggregate all kits with proper references
  kits = {
    # Language Kit
    "language.python" = {
      id          = "language.python"
      type        = "language"
      name        = tofukit_language.python.name
      description = tofukit_language.python.description
      version     = tofukit_language.python.version
      requirements = tofukit_language.python.requirement
    }

    # Tool Kits
    "tool.pip" = {
      id          = "tool.pip"
      type        = "tool"
      name        = tofukit_tool.pip.name
      description = tofukit_tool.pip.description
      version     = tofukit_tool.pip.version
      requirements = tofukit_tool.pip.requirement
    }

    "tool.venv" = {
      id          = "tool.venv"
      type        = "tool"
      name        = tofukit_tool.venv.name
      description = tofukit_tool.venv.description
      version     = tofukit_tool.venv.version
      requirements = tofukit_tool.venv.requirement
    }

    "tool.black" = {
      id          = "tool.black"
      type        = "tool"
      name        = tofukit_tool.black.name
      description = tofukit_tool.black.description
      version     = tofukit_tool.black.version
      requirements = tofukit_tool.black.requirement
    }

    "tool.ruff" = {
      id          = "tool.ruff"
      type        = "tool"
      name        = tofukit_tool.ruff.name
      description = tofukit_tool.ruff.description
      version     = tofukit_tool.ruff.version
      requirements = tofukit_tool.ruff.requirement
    }

    "tool.mypy" = {
      id          = "tool.mypy"
      type        = "tool"
      name        = tofukit_tool.mypy.name
      description = tofukit_tool.mypy.description
      version     = tofukit_tool.mypy.version
      requirements = tofukit_tool.mypy.requirement
    }

    "tool.build" = {
      id          = "tool.build"
      type        = "tool"
      name        = tofukit_tool.build.name
      description = tofukit_tool.build.description
      version     = tofukit_tool.build.version
      requirements = tofukit_tool.build.requirement
    }

    "tool.twine" = {
      id          = "tool.twine"
      type        = "tool"
      name        = tofukit_tool.twine.name
      description = tofukit_tool.twine.description
      version     = tofukit_tool.twine.version
      requirements = tofukit_tool.twine.requirement
    }

    # Framework Kits
    "framework.click" = {
      id          = "framework.click"
      type        = "framework"
      name        = tofukit_framework.click.name
      description = tofukit_framework.click.description
      version     = tofukit_framework.click.version
      requirements = tofukit_framework.click.requirement
    }

    "framework.pytest" = {
      id          = "framework.pytest"
      type        = "framework"
      name        = tofukit_framework.pytest.name
      description = tofukit_framework.pytest.description
      version     = tofukit_framework.pytest.version
      requirements = tofukit_framework.pytest.requirement
    }

    # Methodology Kit
    "methodology.python_standards" = {
      id          = "methodology.python_standards"
      type        = "methodology"
      name        = tofukit_methodology.python_standards.name
      description = tofukit_methodology.python_standards.description
      version     = tofukit_methodology.python_standards.version
      requirements = tofukit_methodology.python_standards.requirement
    }
  }

  # Stack reference
  stack = tofukit_stack.click_app.id

  # Project dependencies
  depends_on = [
    tofukit_language.python,
    tofukit_tool.pip,
    tofukit_tool.venv,
    tofukit_tool.black,
    tofukit_tool.ruff,
    tofukit_tool.mypy,
    tofukit_tool.build,
    tofukit_tool.twine,
    tofukit_framework.click,
    tofukit_framework.pytest,
    tofukit_methodology.python_standards,
    tofukit_stack.click_app
  ]
}

# ========================================
# APPLICATION STACK - Scaffolding & Files
# ========================================
resource "tofukit_stack" "click_app" {
  name        = "click-application-stack"
  description = "Complete Python Click CLI application with testing and packaging"

  depends_on = [
    tofukit_language.python,
    tofukit_framework.click,
    tofukit_framework.pytest
  ]

  # Main CLI Application
  scaffold {
    path = "src/cli.py"
    content = <<-EOF
    #!/usr/bin/env python3
    """
    Deterministic CLI application using Click framework.

    This module provides a command-line interface with multiple
    commands and options for demonstration purposes.
    """
    from typing import Optional, List, Tuple
    import click
    import sys
    import json
    from pathlib import Path


    @click.group()
    @click.version_option(version='1.0.0', prog_name='click-cli')
    @click.pass_context
    def cli(ctx: click.Context) -> None:
        """A deterministic Python CLI application."""
        ctx.ensure_object(dict)
        ctx.obj['config'] = load_config()


    @cli.command()
    @click.option('--name', '-n', default='World', help='Name to greet')
    @click.option('--count', '-c', default=1, type=int, help='Number of greetings')
    @click.option('--shout', is_flag=True, help='Output in uppercase')
    @click.option('--format', type=click.Choice(['text', 'json']), default='text')
    def hello(name: str, count: int, shout: bool, format: str) -> None:
        """Generate greeting messages with various options."""
        for i in range(count):
            greeting = f'Hello, {name}!'
            if shout:
                greeting = greeting.upper()

            if format == 'json':
                output = json.dumps({
                    'iteration': i + 1,
                    'greeting': greeting,
                    'options': {
                        'name': name,
                        'shout': shout
                    }
                })
                click.echo(output)
            else:
                click.echo(greeting)


    @cli.command()
    @click.argument('numbers', nargs=-1, type=float)
    @click.option('--operation', '-o',
                  type=click.Choice(['sum', 'mean', 'min', 'max']),
                  default='sum',
                  help='Mathematical operation to perform')
    def calc(numbers: Tuple[float, ...], operation: str) -> None:
        """Perform calculations on provided numbers."""
        if not numbers:
            click.echo('Error: No numbers provided', err=True)
            sys.exit(1)

        results = {
            'sum': sum(numbers),
            'mean': sum(numbers) / len(numbers),
            'min': min(numbers),
            'max': max(numbers)
        }

        result = results[operation]
        click.echo(f'{operation.capitalize()}: {result}')


    @cli.command()
    @click.option('--input', '-i', type=click.File('r'), default='-',
                  help='Input file (default: stdin)')
    @click.option('--output', '-o', type=click.File('w'), default='-',
                  help='Output file (default: stdout)')
    @click.option('--uppercase', is_flag=True, help='Convert to uppercase')
    @click.option('--lowercase', is_flag=True, help='Convert to lowercase')
    def process(input, output, uppercase: bool, lowercase: bool) -> None:
        """Process text from input to output with transformations."""
        text = input.read()

        if uppercase:
            text = text.upper()
        elif lowercase:
            text = text.lower()

        output.write(text)


    @cli.command()
    @click.pass_context
    def info(ctx: click.Context) -> None:
        """Display application information and configuration."""
        config = ctx.obj.get('config', {})
        info_data = {
            'application': 'Python Click CLI',
            'version': '1.0.0',
            'python_version': sys.version.split()[0],
            'configuration': config
        }
        click.echo(json.dumps(info_data, indent=2))


    def load_config() -> dict:
        """Load configuration from file if exists."""
        config_path = Path.home() / '.clickcli' / 'config.json'
        if config_path.exists():
            with open(config_path, 'r') as f:
                return json.load(f)
        return {'initialized': False}


    if __name__ == '__main__':
        cli()
    EOF
  }

  # Test Suite
  scaffold {
    path = "tests/test_cli.py"
    content = <<-EOF
    """Unit tests for the CLI application."""
    import pytest
    import json
    from click.testing import CliRunner
    from src.cli import cli, hello, calc, process, info


    @pytest.fixture
    def runner():
        """Create a Click test runner."""
        return CliRunner()


    class TestHelloCommand:
        """Test suite for hello command."""

        def test_hello_default(self, runner):
            """Test hello with default parameters."""
            result = runner.invoke(hello)
            assert result.exit_code == 0
            assert "Hello, World!" in result.output

        def test_hello_with_name(self, runner):
            """Test hello with custom name."""
            result = runner.invoke(hello, ['--name', 'Alice'])
            assert result.exit_code == 0
            assert "Hello, Alice!" in result.output

        def test_hello_with_count(self, runner):
            """Test hello with multiple iterations."""
            result = runner.invoke(hello, ['--count', '3'])
            assert result.exit_code == 0
            assert result.output.count("Hello, World!") == 3

        def test_hello_shout(self, runner):
            """Test hello in shout mode."""
            result = runner.invoke(hello, ['--shout'])
            assert result.exit_code == 0
            assert "HELLO, WORLD!" in result.output

        def test_hello_json_format(self, runner):
            """Test hello with JSON output."""
            result = runner.invoke(hello, ['--format', 'json'])
            assert result.exit_code == 0
            data = json.loads(result.output)
            assert data['greeting'] == "Hello, World!"


    class TestCalcCommand:
        """Test suite for calc command."""

        def test_calc_sum(self, runner):
            """Test sum calculation."""
            result = runner.invoke(calc, ['1', '2', '3'])
            assert result.exit_code == 0
            assert "Sum: 6.0" in result.output

        def test_calc_mean(self, runner):
            """Test mean calculation."""
            result = runner.invoke(calc, ['2', '4', '6', '--operation', 'mean'])
            assert result.exit_code == 0
            assert "Mean: 4.0" in result.output

        def test_calc_no_numbers(self, runner):
            """Test calc with no numbers."""
            result = runner.invoke(calc)
            assert result.exit_code == 1
            assert "No numbers provided" in result.output


    class TestProcessCommand:
        """Test suite for process command."""

        def test_process_uppercase(self, runner):
            """Test text uppercase transformation."""
            result = runner.invoke(process, ['--uppercase'], input='hello')
            assert result.exit_code == 0
            assert result.output == 'HELLO'

        def test_process_lowercase(self, runner):
            """Test text lowercase transformation."""
            result = runner.invoke(process, ['--lowercase'], input='HELLO')
            assert result.exit_code == 0
            assert result.output == 'hello'


    class TestInfoCommand:
        """Test suite for info command."""

        def test_info_output(self, runner):
            """Test info command output."""
            result = runner.invoke(cli, ['info'])
            assert result.exit_code == 0
            data = json.loads(result.output)
            assert data['application'] == 'Python Click CLI'
            assert data['version'] == '1.0.0'
    EOF
  }

  # Configuration Files
  scaffold {
    path = "pyproject.toml"
    content = <<-EOF
    [build-system]
    requires = ["setuptools>=68.0", "wheel"]
    build-backend = "setuptools.build_meta"

    [project]
    name = "click-cli"
    version = "1.0.0"
    description = "Deterministic Python CLI application using Click"
    readme = "README.md"
    requires-python = ">=3.8"
    license = {text = "MIT"}
    authors = [
        {name = "Your Name", email = "you@example.com"}
    ]
    classifiers = [
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ]
    dependencies = [
        "click>=8.1.0,<9.0.0",
    ]

    [project.optional-dependencies]
    dev = [
        "pytest>=7.4.0",
        "pytest-cov>=4.1.0",
        "black>=23.11.0",
        "ruff>=0.1.6",
        "mypy>=1.7.0",
        "types-click>=7.1.8",
    ]

    [project.scripts]
    click-cli = "src.cli:cli"

    [tool.setuptools]
    packages = ["src"]

    [tool.black]
    line-length = 88
    target-version = ['py38', 'py39', 'py310', 'py311', 'py312']
    include = '\.pyi?$'

    [tool.ruff]
    target-version = "py38"
    line-length = 88
    select = [
        "E",   # pycodestyle errors
        "W",   # pycodestyle warnings
        "F",   # pyflakes
        "I",   # isort
        "B",   # flake8-bugbear
        "C4",  # flake8-comprehensions
        "UP",  # pyupgrade
    ]
    ignore = ["E501", "B008", "C901"]

    [tool.mypy]
    python_version = "3.8"
    strict = true
    warn_return_any = true
    warn_unused_configs = true
    no_implicit_optional = true
    warn_redundant_casts = true
    warn_unused_ignores = true

    [tool.pytest.ini_options]
    testpaths = ["tests"]
    python_files = "test_*.py"
    addopts = "-v --cov=src --cov-report=term-missing --cov-report=html"
    EOF
  }

  # Requirements Files
  scaffold {
    path = "requirements.txt"
    content = <<-EOF
    click==8.1.7
    EOF
  }

  scaffold {
    path = "requirements-dev.txt"
    content = <<-EOF
    # Production dependencies
    -r requirements.txt

    # Development dependencies
    pytest==7.4.3
    pytest-cov==4.1.0
    black==23.11.0
    ruff==0.1.6
    mypy==1.7.1
    types-click==7.1.8
    build==1.0.3
    twine==4.0.2
    EOF
  }

  # Makefile for Common Tasks
  scaffold {
    path = "Makefile"
    content = <<-EOF
    .PHONY: help install dev test lint format type-check build clean

    help:
    	@echo "Available commands:"
    	@echo "  make install    - Install production dependencies"
    	@echo "  make dev        - Install development dependencies"
    	@echo "  make test       - Run tests with coverage"
    	@echo "  make lint       - Run linting checks"
    	@echo "  make format     - Format code with black"
    	@echo "  make type-check - Run type checking with mypy"
    	@echo "  make build      - Build distribution packages"
    	@echo "  make clean      - Remove build artifacts"

    install:
    	pip install -r requirements.txt

    dev:
    	pip install -r requirements-dev.txt
    	pip install -e .

    test:
    	pytest

    lint:
    	ruff check src tests

    format:
    	black src tests

    type-check:
    	mypy src

    build: clean
    	python -m build

    clean:
    	rm -rf build dist *.egg-info
    	rm -rf .pytest_cache .coverage htmlcov
    	find . -type d -name __pycache__ -exec rm -rf {} +
    	find . -type f -name "*.pyc" -delete
    EOF
  }

  # Docker Support
  scaffold {
    path = "Dockerfile"
    content = <<-EOF
    FROM python:3.12-slim

    WORKDIR /app

    COPY requirements.txt .
    RUN pip install --no-cache-dir -r requirements.txt

    COPY src/ ./src/
    COPY pyproject.toml .

    RUN pip install --no-cache-dir -e .

    ENTRYPOINT ["click-cli"]
    CMD ["--help"]
    EOF
  }

  # CI/CD Configuration
  scaffold {
    path = ".github/workflows/ci.yml"
    content = <<-EOF
    name: CI

    on:
      push:
        branches: [main, develop]
      pull_request:
        branches: [main]

    jobs:
      test:
        runs-on: ubuntu-latest
        strategy:
          matrix:
            python-version: ['3.8', '3.9', '3.10', '3.11', '3.12']

        steps:
        - uses: actions/checkout@v3
        - name: Set up Python ${{ matrix.python-version }}
          uses: actions/setup-python@v4
          with:
            python-version: ${{ matrix.python-version }}

        - name: Install dependencies
          run: |
            python -m pip install --upgrade pip
            pip install -r requirements-dev.txt
            pip install -e .

        - name: Lint with ruff
          run: ruff check src tests

        - name: Format check with black
          run: black --check src tests

        - name: Type check with mypy
          run: mypy src

        - name: Test with pytest
          run: pytest --cov=src --cov-report=xml

        - name: Upload coverage to Codecov
          uses: codecov/codecov-action@v3
          with:
            file: ./coverage.xml
    EOF
  }

  # README
  scaffold {
    path = "README.md"
    content = <<-EOF
    # Python Click CLI Application

    A deterministic command-line application built with Python and Click framework, showcasing best practices and multiple tool kits.

    ## Features

    - Multiple CLI commands (hello, calc, process, info)
    - Comprehensive test coverage
    - Type hints and static typing
    - Code formatting with Black
    - Linting with Ruff
    - Package distribution ready
    - Docker support
    - CI/CD pipeline

    ## Installation

    ### Development Setup

    1. Create and activate virtual environment:
    ```bash
    python3 -m venv .venv
    source .venv/bin/activate  # On Windows: .venv\Scripts\activate
    ```

    2. Install development dependencies:
    ```bash
    make dev
    ```

    ### Production Installation

    ```bash
    pip install click-cli
    ```

    ## Usage

    ### Hello Command
    ```bash
    # Basic greeting
    click-cli hello

    # Custom name with repetition
    click-cli hello --name Alice --count 3

    # Shout mode
    click-cli hello --shout

    # JSON output
    click-cli hello --format json
    ```

    ### Calculator Command
    ```bash
    # Sum numbers
    click-cli calc 1 2 3 4 5

    # Calculate mean
    click-cli calc 10 20 30 --operation mean

    # Find maximum
    click-cli calc 5 12 3 8 --operation max
    ```

    ### Process Command
    ```bash
    # Convert to uppercase
    echo "hello world" | click-cli process --uppercase

    # Process file
    click-cli process -i input.txt -o output.txt --lowercase
    ```

    ### Info Command
    ```bash
    # Display application info
    click-cli info
    ```

    ## Development

    ### Running Tests
    ```bash
    make test
    ```

    ### Code Quality
    ```bash
    # Format code
    make format

    # Run linter
    make lint

    # Type checking
    make type-check
    ```

    ### Building Package
    ```bash
    make build
    ```

    ## Docker

    Build and run with Docker:
    ```bash
    docker build -t click-cli .
    docker run --rm click-cli hello --name Docker
    ```

    ## Project Structure

    ```
    .
    ├── src/
    │   └── cli.py           # Main CLI application
    ├── tests/
    │   └── test_cli.py      # Test suite
    ├── pyproject.toml       # Project configuration
    ├── requirements.txt     # Production dependencies
    ├── requirements-dev.txt # Development dependencies
    ├── Makefile            # Common tasks
    ├── Dockerfile          # Container definition
    └── README.md           # Documentation
    ```

    ## License

    MIT License
    EOF
  }

  # Git configuration
  scaffold {
    path = ".gitignore"
    content = <<-EOF
    # Python
    __pycache__/
    *.py[cod]
    *$py.class
    *.so
    .Python
    env/
    venv/
    .venv/
    ENV/
    env.bak/
    venv.bak/

    # Distribution
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
    MANIFEST

    # Testing
    .tox/
    .coverage
    .coverage.*
    .cache
    .pytest_cache/
    nosetests.xml
    coverage.xml
    *.cover
    .hypothesis/
    htmlcov/

    # Type checking
    .mypy_cache/
    .dmypy.json
    dmypy.json
    .ruff_cache/

    # IDEs
    .idea/
    .vscode/
    *.swp
    *.swo
    *~
    .DS_Store

    # Project specific
    .tofukit/
    *.log
    EOF
  }
}

# ========================================
# OUTPUT DEFINITIONS
# ========================================
output "project_info" {
  value = {
    name        = tofukit_project.click_cli.name
    version     = tofukit_project.click_cli.version
    description = tofukit_project.click_cli.description
  }
  description = "Project information"
}

output "kit_summary" {
  value = {
    total_kits = length(tofukit_project.click_cli.kits)
    kit_types = {
      languages   = 1
      tools       = 7
      frameworks  = 2
      methodology = 1
    }
  }
  description = "Summary of kits used in the project"
}

output "quick_start" {
  value = <<-EOT
  Quick Start Commands:
    1. terraform init
    2. terraform apply
    3. cd <project_dir>
    4. python3 -m venv .venv
    5. source .venv/bin/activate
    6. make dev
    7. click-cli --help
  EOT
  description = "Quick start guide"
}