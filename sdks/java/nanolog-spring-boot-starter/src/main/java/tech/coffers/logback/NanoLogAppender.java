package tech.coffers.logback;

import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.AppenderBase;
import lombok.Getter;
import lombok.Setter;

import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.InetAddress;
import java.net.UnknownHostException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Async Logback appender that ships logs to NanoLog server.
 * Uses a background worker thread with batching for high performance.
 * 
 * <p>
 * Features:
 * <ul>
 * <li>Exponential backoff retry (100ms -> 200ms -> 400ms)</li>
 * <li>Local fallback when server is unavailable</li>
 * <li>Automatic recovery when server becomes available</li>
 * </ul>
 */
@Getter
@Setter
public class NanoLogAppender extends AppenderBase<ILoggingEvent> {

    private static final int MAX_RETRIES = 3;
    private static final long INITIAL_BACKOFF_MS = 100;
    private static final long RECOVERY_INTERVAL_MS = 60_000; // 1 minute

    private String serverUrl = "http://localhost:8080";
    private String serviceName = "default";
    private int batchSize = 100;
    private long flushIntervalMs = 1000;
    private String host = "unknown";

    // Fallback configuration
    private boolean enableFallback = true;
    private String fallbackPath = "/tmp/nanolog/fallback";

    private final LinkedBlockingQueue<ILoggingEvent> queue = new LinkedBlockingQueue<>(10000);
    private final AtomicBoolean running = new AtomicBoolean(false);
    private Thread workerThread;
    private Thread recoveryThread;

    @Override
    public void start() {
        if (isStarted()) {
            return;
        }
        super.start();
        running.set(true);

        // Start worker thread
        workerThread = new Thread(this::workerLoop, "NanoLog-Worker");
        workerThread.setDaemon(true);
        workerThread.start();

        // Start recovery thread if fallback is enabled
        if (enableFallback) {
            recoveryThread = new Thread(this::recoveryLoop, "NanoLog-Recovery");
            recoveryThread.setDaemon(true);
            recoveryThread.start();
        }

        try {
            if ("unknown".equals(host)) {
                host = InetAddress.getLocalHost().getHostName();
            }
        } catch (UnknownHostException e) {
            // keep unknown
        }

        addInfo("NanoLogAppender started. Server: " + serverUrl + ", Service: " + serviceName +
                ", Host: " + host + ", Fallback: " + (enableFallback ? fallbackPath : "disabled"));
    }

    @Override
    public void stop() {
        if (!isStarted()) {
            return;
        }
        running.set(false);

        // Stop worker thread
        if (workerThread != null) {
            workerThread.interrupt();
            try {
                workerThread.join(5000);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }

        // Stop recovery thread
        if (recoveryThread != null) {
            recoveryThread.interrupt();
            try {
                recoveryThread.join(2000);
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

        String json = buildJsonPayload(batch);
        boolean success = sendBatchWithRetry(json);

        if (!success) {
            writeToFallback(json);
        }
    }

    /**
     * Attempts to send a JSON payload with exponential backoff retry.
     * Retry delays: 100ms -> 200ms -> 400ms
     *
     * @param json the JSON payload to send
     * @return true if successful, false if all retries failed
     */
    private boolean sendBatchWithRetry(String json) {
        long backoff = INITIAL_BACKOFF_MS;

        for (int attempt = 1; attempt <= MAX_RETRIES; attempt++) {
            try {
                postToServer(json);
                return true; // Success
            } catch (Exception e) {
                addWarn("NanoLog send attempt " + attempt + "/" + MAX_RETRIES + " failed: " + e.getMessage());
                if (attempt < MAX_RETRIES) {
                    try {
                        Thread.sleep(backoff);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        return false;
                    }
                    backoff *= 2; // Exponential backoff: 100ms -> 200ms -> 400ms
                }
            }
        }
        return false; // All retries failed
    }

    /**
     * Writes failed logs to local fallback file.
     * Each JSON batch is written as a single line.
     *
     * @param json the JSON payload to write
     */
    private void writeToFallback(String json) {
        if (!enableFallback) {
            addWarn("Fallback disabled, dropping batch");
            return;
        }

        try {
            Path fallbackDir = Paths.get(fallbackPath);
            Files.createDirectories(fallbackDir);
            Path file = fallbackDir.resolve("fallback.log");

            // Append write, one JSON per line
            Files.write(file, (json + "\n").getBytes(StandardCharsets.UTF_8),
                    StandardOpenOption.CREATE, StandardOpenOption.APPEND);
            addInfo("Wrote batch to fallback: " + file);
        } catch (IOException e) {
            addError("Failed to write fallback: " + e.getMessage(), e);
        }
    }

    /**
     * Recovery loop that runs every minute to resend failed logs.
     */
    private void recoveryLoop() {
        while (running.get()) {
            try {
                Thread.sleep(RECOVERY_INTERVAL_MS);
                tryRecoverFallback();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            }
        }
    }

    /**
     * Attempts to recover and resend logs from the fallback file.
     * Only proceeds if the server is reachable (ping succeeds).
     */
    private void tryRecoverFallback() {
        Path file = Paths.get(fallbackPath, "fallback.log");
        if (!Files.exists(file)) {
            return;
        }

        // First check if server is available
        if (!pingServer()) {
            addInfo("Server not available, skipping recovery");
            return;
        }

        try {
            List<String> lines = Files.readAllLines(file, StandardCharsets.UTF_8);
            if (lines.isEmpty()) {
                Files.deleteIfExists(file);
                return;
            }

            addInfo("Starting recovery of " + lines.size() + " batches from fallback");
            List<String> failedLines = new ArrayList<>();

            for (String json : lines) {
                if (json.trim().isEmpty()) {
                    continue;
                }
                if (!sendBatchWithRetry(json)) {
                    // Re-add to failed lines if still can't send
                    failedLines.add(json);
                }
            }

            if (failedLines.isEmpty()) {
                Files.delete(file);
                addInfo("Recovery completed successfully, fallback file deleted");
            } else {
                // Rewrite failed lines
                Files.write(file, failedLines, StandardCharsets.UTF_8,
                        StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING);
                addWarn("Partial recovery: " + (lines.size() - failedLines.size()) +
                        " succeeded, " + failedLines.size() + " still pending");
            }
        } catch (IOException e) {
            addError("Recovery failed: " + e.getMessage(), e);
        }
    }

    /**
     * Pings the server to check availability.
     *
     * @return true if server responds with 200, false otherwise
     */
    private boolean pingServer() {
        try {
            URL url = new URL(serverUrl + "/api/ping");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(2000);
            conn.setReadTimeout(2000);
            conn.setRequestMethod("GET");
            int code = conn.getResponseCode();
            conn.disconnect();
            return code == 200;
        } catch (Exception e) {
            return false;
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
            sb.append("\"host\":\"").append(escapeJson(host)).append("\",");
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
                throw new IOException("Server returned HTTP " + responseCode);
            }
        } finally {
            conn.disconnect();
        }
    }
}
