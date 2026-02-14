#!/usr/bin/env python3
"""
Main application entry point for test-project.

This module serves as the primary entry point for the application,
coordinating between configuration, utilities, and core functionality.
"""

import sys
from config import Config
from utils import setup_logging, validate_environment


def main():
    """Main application function."""
    # Setup logging
    logger = setup_logging()
    logger.info("Starting test-project v1.0.0")

    # Load configuration
    config = Config()
    logger.info(f"Loaded configuration: {config.app_name}")

    # Validate environment
    if not validate_environment():
        logger.error("Environment validation failed")
        sys.exit(1)

    logger.info("Application initialized successfully")

    # Main application logic would go here
    print(f"Welcome to {config.app_name}!")
    print(f"Environment: {config.environment}")
    print(f"Debug mode: {config.debug}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
