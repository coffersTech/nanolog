# NanoLog Python SDK

Official Python SDK for NanoLog.

## Installation

```bash
pip install nanolog-sdk
```

## Usage

```python
from nanolog import NanoLogHandler
import logging

logger = logging.getLogger("my_app")
logger.addHandler(NanoLogHandler(
    server_url="http://localhost:8088",
    api_key="sk-xxxx",
    service="my-service"
))

logger.info("Hello World")
```
