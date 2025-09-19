# Click CLI Hello World Example

This example demonstrates how to use the TofuKit provider to define a complete Python CLI application using the Click framework.

## What This Example Includes

1. **Language Definition**: Python 3.12.6
2. **Tools**: pip package manager and venv for virtual environments
3. **Framework**: Click for CLI creation
4. **Methodology**: Idiomatic Python practices
5. **Stack**: Complete Python Click CLI development stack
6. **Project**: Hello World CLI with comprehensive requirements
7. **Scaffolding**: All necessary files for a production-ready CLI app

## Files Created by the Stack

- `app.py` - Main application with Click decorators
- `requirements.txt` - Production dependencies
- `requirements-dev.txt` - Development dependencies
- `setup.py` - Package configuration for pip installation
- `README.md` - Project documentation
- `.gitignore` - Git ignore patterns for Python
- `test_app.py` - Unit tests using pytest

## Running This Example

1. Navigate to this directory:
   ```bash
   cd provider/examples/click-cli-hello-world
   ```

2. Set up the provider override:
   ```bash
   export TF_CLI_CONFIG_FILE=/Users/dmitry/dev/kirr/kirr.dev/apps/tofukit/.tofurc
   ```

3. Initialize OpenTofu:
   ```bash
   tofu init
   ```

4. Plan the configuration:
   ```bash
   tofu plan
   ```

5. Apply the configuration:
   ```bash
   tofu apply
   ```

6. View the outputs:
   ```bash
   tofu output -json project_context
   ```

## Generated Application Features

The scaffolded CLI application includes:

- **Basic greeting**: `python app.py` → "Hello, World!"
- **Custom name**: `python app.py --name Alice` → "Hello, Alice!"
- **Repeat count**: `python app.py --count 3` → prints 3 times
- **Shout mode**: `python app.py --shout` → "HELLO, WORLD!"
- **Version info**: `python app.py --version`
- **Help text**: `python app.py --help`

## Testing the Generated App

After applying the configuration, you can test the generated application:

```bash
# Create the app files (would normally be done by the provider)
# For now, manually create app.py with the content from the scaffold

# Set up the environment
python3 -m venv .venv
source .venv/bin/activate

# Install dependencies
pip install click

# Run the app
python app.py
python app.py --name "TofuKit"
python app.py --count 3 --shout

# Run tests (after installing pytest)
pip install pytest
pytest test_app.py
```

## Project Requirements

The project defines comprehensive requirements including:

1. Project setup and initialization
2. Basic Hello World functionality
3. Command-line parameters (name, count, shout)
4. Version and help information
5. Unit testing with pytest
6. Code quality (Black, mypy, ruff)
7. Documentation
8. Packaging for distribution

## Customization

You can modify this example to:

- Change the Python version
- Add more dependencies
- Modify the scaffolded files
- Add additional requirements
- Include more methodologies or styles

## Output for LLMs

The configuration generates structured output suitable for LLMs, including:

- Complete project metadata
- All component dependencies
- File scaffolding templates
- Setup and test commands
- Requirements with verification steps

This makes it easy to feed the entire project context to an LLM for code generation or assistance.
