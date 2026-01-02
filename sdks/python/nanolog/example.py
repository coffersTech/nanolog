import logging
import time
import sys
import os

# Add the parent directory to sys.path to import nanolog
sys.path.append(os.path.join(os.path.dirname(__file__), '../../'))

from sdks.python.nanolog.handler import NanoLogHandler

def main():
    logger = logging.getLogger("test_logger")
    logger.setLevel(logging.INFO)
    
    # Configure NanoLog Handler
    # Assuming server is running on localhost:8088
    try:
        handler = NanoLogHandler(
            server_url="http://localhost:8088", 
            api_key="sk-dev-test-key", 
            service="python-example-script"
        )
        logger.addHandler(handler)
        print("NanoLog Handler added.")
    except Exception as e:
        print(f"Failed to initialize NanoLog Handler: {e}")
        return

    # Generate some logs
    print("Generating logs...")
    for i in range(5):
        logger.info(f"Log message {i} from Python SDK", {"iteration": i, "custom_tag": "example"})
        time.sleep(0.5)
    
    logger.warning("This is a warning message!")
    logger.error("This is an error message!", {"error_code": 500})
    
    print("Logs generated. Waiting for worker to flush...")
    # Give the worker a moment to send remaining logs (though atexit should handle it)
    time.sleep(2)
    print("Done.")

if __name__ == "__main__":
    main()
