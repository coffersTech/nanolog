package tech.coffers.logback;

import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.AppenderBase;
import lombok.Getter;
import lombok.Setter;

import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Async Logback appender that ships logs to NanoLog server.
 * Uses a background worker thread with batching for high performance.
 */
@Getter
@Setter
public class NanoLogAppender extends AppenderBase<ILoggingEvent> {

    private String serverUrl = "http://localhost:8080";
    private String serviceName = "default";
    private int batchSize = 100;
    private long flushIntervalMs = 1000;

    private final LinkedBlockingQueue<ILoggingEvent> queue = new LinkedBlockingQueue<>(10000);
    private final AtomicBoolean running = new AtomicBoolean(false);
    private Thread workerThread;

    @Override
    public void start() {
        if (isStarted()) {
            return;
        }
        super.start();
        running.set(true);
        workerThread = new Thread(this::workerLoop, "NanoLog-Worker");
        workerThread.setDaemon(true);
        workerThread.start();
        addInfo("NanoLogAppender started. Server: " + serverUrl + ", Service: " + serviceName);
    }

    @Override
    public void stop() {
        if (!isStarted()) {
            return;
        }
        running.set(false);
        if (workerThread != null) {
            workerThread.interrupt();
            try {
                workerThread.join(5000);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }
        // Flush remaining logs
        flushQueue();
        super.stop();
        addInfo("NanoLogAppender stopped.");
    }

    @Override
    protected void append(ILoggingEvent event) {
        // Never block the logging thread
        if (!queue.offer(event)) {
            addWarn("NanoLog queue full, dropping log event");
        }
    }

    private void workerLoop() {
        List<ILoggingEvent> batch = new ArrayList<>(batchSize);
        long lastFlush = System.currentTimeMillis();

        while (running.get() || !queue.isEmpty()) {
            try {
                ILoggingEvent event = queue.poll(100, TimeUnit.MILLISECONDS);
                if (event != null) {
                    batch.add(event);
                }

                long now = System.currentTimeMillis();
                boolean shouldFlush = batch.size() >= batchSize ||
                        (now - lastFlush >= flushIntervalMs && !batch.isEmpty());

                if (shouldFlush) {
                    sendBatch(batch);
                    batch.clear();
                    lastFlush = now;
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            }
        }

        // Final flush on shutdown
        if (!batch.isEmpty()) {
            sendBatch(batch);
        }
    }

    private void flushQueue() {
        List<ILoggingEvent> remaining = new ArrayList<>();
        queue.drainTo(remaining);
        if (!remaining.isEmpty()) {
            sendBatch(remaining);
        }
    }

    private void sendBatch(List<ILoggingEvent> batch) {
        if (batch.isEmpty()) {
            return;
        }

        try {
            String json = buildJsonPayload(batch);
            postToServer(json);
        } catch (Exception e) {
            addError("Failed to send batch to NanoLog server: " + e.getMessage(), e);
        }
    }

    private String buildJsonPayload(List<ILoggingEvent> events) {
        StringBuilder sb = new StringBuilder();
        sb.append("[");
        for (int i = 0; i < events.size(); i++) {
            if (i > 0)
                sb.append(",");
            ILoggingEvent e = events.get(i);
            sb.append("{");
            sb.append("\"timestamp\":").append(e.getTimeStamp() * 1_000_000).append(","); // Convert to nanos
            sb.append("\"level\":\"").append(mapLevel(e.getLevel().levelInt)).append("\",");
            sb.append("\"service\":\"").append(escapeJson(serviceName)).append("\",");
            sb.append("\"message\":\"").append(escapeJson(e.getFormattedMessage())).append("\"");
            sb.append("}");
        }
        sb.append("]");
        return sb.toString();
    }

    private String mapLevel(int levelInt) {
        // Logback levels: TRACE=5000, DEBUG=10000, INFO=20000, WARN=30000, ERROR=40000
        if (levelInt >= 40000)
            return "3"; // ERROR
        if (levelInt >= 30000)
            return "2"; // WARN
        if (levelInt >= 20000)
            return "1"; // INFO
        return "0"; // DEBUG/TRACE
    }

    private String escapeJson(String s) {
        if (s == null)
            return "";
        return s.replace("\\", "\\\\")
                .replace("\"", "\\\"")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                .replace("\t", "\\t");
    }

    private void postToServer(String json) throws Exception {
        URL url = new URL(serverUrl + "/api/ingest");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        try {
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setDoOutput(true);
            conn.setConnectTimeout(5000);
            conn.setReadTimeout(5000);

            try (OutputStream os = conn.getOutputStream()) {
                os.write(json.getBytes(StandardCharsets.UTF_8));
            }

            int responseCode = conn.getResponseCode();
            if (responseCode != 200) {
                addWarn("NanoLog server returned: " + responseCode);
            }
        } finally {
            conn.disconnect();
        }
    }
}
