import logging
import queue
import atexit
import os
import uuid
import json
import socket
import time
import urllib.request
import urllib.error
import sys
from pathlib import Path
from .worker import Worker

class NanoLogHandler(logging.Handler):
    def __init__(self, server_url, api_key, service, source_host=None):
        super().__init__()
        self.server_url = server_url.rstrip('/')
        self.api_key = api_key
        self.service = service
        self.source_host = source_host or socket.gethostname()
        
        self.queue = queue.Queue(maxsize=10000)
        self.instance_id = self._get_or_create_instance_id()
        
        self._register_instance()
        
        self.worker = Worker(self.queue, self.server_url, self.api_key, self.instance_id)
        self.worker.start()
        
        atexit.register(self.shutdown)

    def _get_or_create_instance_id(self):
        home = Path.home()
        nanolog_dir = home / ".nanolog"
        nanolog_dir.mkdir(exist_ok=True)
        id_file = nanolog_dir / "id"
        
        if id_file.exists():
            try:
                return id_file.read_text().strip()
            except Exception:
                pass
        
        new_id = str(uuid.uuid4())
        try:
            id_file.write_text(new_id)
        except Exception:
            pass # Use ephemeral ID if write fails
            
        return new_id

    def _register_instance(self):
        # Handshake with server
        try:
            data = json.dumps({
                "instance_id": self.instance_id,
                "service_name": self.service,
                "host_name": self.source_host,
                "platform": "python",
                "version": "0.1.0"
            }).encode('utf-8')

            req = urllib.request.Request(
                f"{self.server_url}/api/registry/handshake",
                data=data,
                method='POST'
            )
            req.add_header('Content-Type', 'application/json')
            req.add_header('Authorization', f'Bearer {self.api_key}')
            
            with urllib.request.urlopen(req) as response:
                 if response.status != 200:
                    sys.stderr.write(f"NanoLog Handshake Failed: HTTP {response.status}\n")

        except Exception as e:
            sys.stderr.write(f"NanoLog Handshake Error: {e}\n")

    def emit(self, record):
        try:
            # Capture standard logging attributes
            msg = self.format(record)
            
            log_entry = {
                "timestamp": int(record.created * 1000000000), # Nanoseconds
                "level": record.levelname,
                "message": msg,
                "logger": record.name,
                "thread": record.threadName,
                "file": record.filename,
                "line": record.lineno,
                "service": self.service,
                "host": self.source_host,
                "instance_id": self.instance_id,
                "attributes": {}
            }
            
            # Handle extra fields (args) if they are a dict
            if isinstance(record.args, dict):
                 log_entry["attributes"].update(record.args)
                 
            # Trying to extract common trace info if present in record
            if hasattr(record, 'trace_id'):
                 log_entry["trace_id"] = str(record.trace_id)
            if hasattr(record, 'span_id'):
                 log_entry["span_id"] = str(record.span_id)

            try:
                self.queue.put_nowait(log_entry)
            except queue.Full:
                sys.stderr.write("NanoLog Queue Full: Dropping log\n")

        except Exception:
            self.handleError(record)

    def shutdown(self):
        # Stop worker
        if hasattr(self, 'worker') and self.worker.is_alive():
            self.worker.stop()
            self.worker.join()
