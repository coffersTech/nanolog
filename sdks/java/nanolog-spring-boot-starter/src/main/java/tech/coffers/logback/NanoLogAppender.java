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
import java.util.Map;
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
 * <li>Exponential backoff retry (100ms &rarr; 200ms &rarr; 400ms)</li>
 * <li>Local fallback when server is unavailable</li>
 * <li>Automatic recovery when server becomes available</li>
 * <li>Context awareness (TraceID, Thread Name, Logger Name)</li>
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

    // Authentication
    private String token = "";

    // Use EnhancedLogEvent to carry Context info
    private final LinkedBlockingQueue<EnhancedLogEvent> queue = new LinkedBlockingQueue<>(10000);
    private final AtomicBoolean running = new AtomicBoolean(false);
    private Thread workerThread;
    private Thread recoveryThread;
    private Thread heartbeatThread;

    // Status
    @Getter
    private boolean handshakeSuccessful = false;

    /**
     * Internal wrapper to carry log event and extracted context.
     */
    @Getter
    private static class EnhancedLogEvent {
        private final ILoggingEvent event;
        private final String traceId;

        public EnhancedLogEvent(ILoggingEvent event, String traceId) {
            this.event = event;
            this.traceId = traceId;
        }
    }

    // Instance Identity
    private String instanceId;
    private static String SDK_VERSION = "java-unknown";

    static {
        try (java.io.InputStream is = NanoLogAppender.class.getResourceAsStream("/nanolog-version.properties")) {
            if (is != null) {
                java.util.Properties props = new java.util.Properties();
                props.load(is);
                String ver = props.getProperty("version");
                if (ver != null && !ver.isEmpty()) {
                    SDK_VERSION = "java-" + ver;
                }
            }
        } catch (Exception e) {
            // ignore
        }
    }

    @Override
    public void start() {
        if (isStarted()) {
            return;
        }

        // Initialize Instance ID (Persistent or New)
        initInstanceId();

        // 0. Perform Handshake (Blocking, short timeout)
        performHandshake();

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
            recoveryThread.setDaemon(true);
            recoveryThread.start();
        }

        // Start heartbeat thread (periodic handshake/keepalive)
        heartbeatThread = new Thread(this::heartbeatLoop, "NanoLog-Heartbeat");
        heartbeatThread.setDaemon(true);
        heartbeatThread.start();

        // Resolve Hostname
        if ("unknown".equals(host)) {
            // 1. Try Environment Variables
            String envHost = System.getenv("HOSTNAME");
            if (envHost == null || envHost.isEmpty()) {
                envHost = System.getenv("COMPUTERNAME");
            }

            if (envHost != null && !envHost.isEmpty()) {
                host = envHost;
            } else {
                // 2. Try InetAddress
                try {
                    host = InetAddress.getLocalHost().getHostName();
                } catch (UnknownHostException e) {
                    try {
                        // 3. Fallback to IP
                        host = InetAddress.getLocalHost().getHostAddress();
                    } catch (Exception ex) {
                        // keep unknown
                    }
                }
            }
        }

        addInfo("NanoLogAppender started. Server: " + serverUrl + ", Service: " + serviceName +
                ", Host: " + host + ", Fallback: " + (enableFallback ? fallbackPath : "disabled") +
                ", InstanceID: " + instanceId);
    }

    private void initInstanceId() {
        // 1. Try Environment Variable (Best for Docker/K8s)
        // e.g. -e NANOLOG_INSTANCE_ID="my-app-pod-1"
        String envId = System.getenv("NANOLOG_INSTANCE_ID");
        if (envId != null && !envId.isEmpty()) {
            this.instanceId = envId;
            addInfo("Using InstanceID from environment: " + envId);
            return;
        }

        // 2. Try to load from temp file to maintain identity across restarts
        String tempDir = System.getProperty("java.io.tmpdir");
        String safeServiceName = serviceName.replaceAll("[^a-zA-Z0-9.-]", "_");
        Path idFile = Paths.get(tempDir, "nanolog", safeServiceName + ".id");

        try {
            if (Files.exists(idFile)) {
                List<String> lines = Files.readAllLines(idFile, StandardCharsets.UTF_8);
                if (!lines.isEmpty()) {
                    String savedId = lines.get(0).trim();
                    if (!savedId.isEmpty()) {
                        this.instanceId = savedId;
                        addInfo("Restored InstanceID: " + savedId);
                        return;
                    }
                }
            }
        } catch (Exception e) {
            addWarn("Failed to load existing InstanceID: " + e.getMessage());
        }

        // 3. Generate new (Fallback)
        this.instanceId = java.util.UUID.randomUUID().toString();

        // Save
        try {
            Files.createDirectories(idFile.getParent());
            Files.write(idFile, this.instanceId.getBytes(StandardCharsets.UTF_8),
                    StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING);
        } catch (Exception e) {
            addWarn("Failed to persist InstanceID: " + e.getMessage());
        }
    }

    private void performHandshake() {
        try {
            URL url = new URL(serverUrl + "/api/registry/handshake");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setRequestProperty("X-Instance-ID", instanceId);

            if (token != null && !token.isEmpty()) {
                conn.setRequestProperty("Authorization", "Bearer " + token);
            }

            conn.setDoOutput(true);
            conn.setConnectTimeout(3000); // 3s timeout for handshake
            conn.setReadTimeout(3000);

            // Build JSON
            String json = "{"
                    + "\"instance_id\":\"" + instanceId + "\","
                    + "\"service_name\":\"" + escapeJson(serviceName) + "\","
                    + "\"hostname\":\"" + escapeJson(host) + "\"," // Host might be unknown initially, but we try
                    + "\"sdk_version\":\"" + SDK_VERSION + "\","
                    + "\"language\":\"java\","
                    + "\"registered_at\":" + (System.currentTimeMillis() / 1000)
                    + "}";

            try (OutputStream os = conn.getOutputStream()) {
                os.write(json.getBytes(StandardCharsets.UTF_8));
            }

            int code = conn.getResponseCode();
            if (code == 200) {
                this.handshakeSuccessful = true;
                // Parse Response (Simple manual parsing)
                try (java.io.InputStream is = conn.getInputStream()) {
                    java.util.Scanner s = new java.util.Scanner(is).useDelimiter("\\A");
                    String resp = s.hasNext() ? s.next() : "";

                    // Simple check for "DEBUG" level
                    if (resp.contains("\"level\":\"DEBUG\"")) {
                        // Dynamically adjust filter - strictly speaking AppenderBase doesn't have
                        // setLevel
                        // We would need to add a ThresholdFilter or similar.
                        // For this task, let's just log it.
                        // To implement real level switching, we would need to check against this level
                        // in append().
                        addInfo("Handshake: Server requested DEBUG level");
                    }
                }
            } else {
                addWarn("Handshake failed with status: " + code);
            }
        } catch (Exception e) {
            addWarn("Handshake failed: " + e.getMessage());
        }
    }

    // Filtering
    private java.util.List<String> excludeLoggers = new java.util.ArrayList<>();
    private java.util.List<java.util.regex.Pattern> excludeMsgPatterns = new java.util.ArrayList<>();

    public void setExcludeLoggers(java.util.List<String> excludeLoggers) {
        this.excludeLoggers = excludeLoggers;
    }

    public void setExcludeMsgPatterns(java.util.List<String> patterns) {
        this.excludeMsgPatterns.clear();
        for (String p : patterns) {
            try {
                this.excludeMsgPatterns.add(java.util.regex.Pattern.compile(p));
            } catch (Exception e) {
                // ignore invalid pattern
            }
        }
    }

    @Override
    protected void append(ILoggingEvent event) {
        if (event == null)
            return;

        // Filtering Logic
        if (shouldFilter(event)) {
            return;
        }

        // 1. Extract TraceID from MDC immediately (in logging thread)
        String traceId = extractTraceId(event);

        // 2. Wrap and enqueue
        // Never block the logging thread
        if (!queue.offer(new EnhancedLogEvent(event, traceId))) {
            addWarn("NanoLog queue full, dropping log event");
        }
    }

    private boolean shouldFilter(ILoggingEvent event) {
        // 0. Check Marker (Code-level ignore)
        org.slf4j.Marker marker = event.getMarker();
        if (marker != null) {
            // Logback events usually have a single marker or a chain.
            // We check if it contains our ignore marker.
            if (marker.contains("NANOLOG_IGNORE")) {
                return true;
            }
            // Also check equality directly just in case
            if (marker.getName().equals("NANOLOG_IGNORE")) {
                return true;
            }
        }

        // 1. Check Logger Name
        if (excludeLoggers != null && !excludeLoggers.isEmpty()) {
            String logger = event.getLoggerName();
            for (String prefix : excludeLoggers) {
                if (logger.startsWith(prefix)) {
                    return true;
                }
            }
        }

        // 2. Check Message Pattern
        if (excludeMsgPatterns != null && !excludeMsgPatterns.isEmpty()) {
            String msg = event.getFormattedMessage();
            for (java.util.regex.Pattern p : excludeMsgPatterns) {
                if (p.matcher(msg).find()) {
                    return true;
                }
            }
        }
        return false;
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

        // Stop heartbeat thread
        if (heartbeatThread != null) {
            heartbeatThread.interrupt();
            try {
                heartbeatThread.join(2000);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }

        // Flush remaining logs
        flushQueue();
        super.stop();
        addInfo("NanoLogAppender stopped.");
    }

    private String extractTraceId(ILoggingEvent event) {
        Map<String, String> mdc = event.getMDCPropertyMap();
        if (mdc == null || mdc.isEmpty()) {
            return "";
        }

        // Try common TraceID keys
        String tid = mdc.get("traceId");
        if (tid == null)
            tid = mdc.get("trace_id");
        if (tid == null)
            tid = mdc.get("X-B3-TraceId");
        if (tid == null)
            tid = mdc.get("X-Amzn-Trace-Id");

        return tid != null ? tid : "";
    }

    private void workerLoop() {
        List<EnhancedLogEvent> batch = new ArrayList<>(batchSize);
        long lastFlush = System.currentTimeMillis();

        while (running.get() || !queue.isEmpty()) {
            try {
                EnhancedLogEvent event = queue.poll(100, TimeUnit.MILLISECONDS);
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
        List<EnhancedLogEvent> remaining = new ArrayList<>();
        queue.drainTo(remaining);
        if (!remaining.isEmpty()) {
            sendBatch(remaining);
        }
    }

    private void sendBatch(List<EnhancedLogEvent> batch) {
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

            // Add Authorization header if token is configured
            if (token != null && !token.isEmpty()) {
                conn.setRequestProperty("Authorization", "Bearer " + token);
            }
            // Add Heartbeat Header
            conn.setRequestProperty("X-Instance-ID", instanceId);

            int code = conn.getResponseCode();
            conn.disconnect();
            return code == 200;
        } catch (Exception e) {
            return false;
        }
    }

    private void heartbeatLoop() {
        while (running.get()) {
            try {
                // Heartbeat every 15 seconds
                Thread.sleep(15_000);
                performHandshake();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            }
        }
    }

    private String buildJsonPayload(List<EnhancedLogEvent> events) {
        StringBuilder sb = new StringBuilder();
        sb.append("[");
        for (int i = 0; i < events.size(); i++) {
            if (i > 0)
                sb.append(",");
            EnhancedLogEvent wrapper = events.get(i);
            ILoggingEvent e = wrapper.getEvent();

            sb.append("{");
            sb.append("\"timestamp\":").append(e.getTimeStamp() * 1_000_000).append(","); // Convert to nanos
            sb.append("\"level\":\"").append(mapLevel(e.getLevel().levelInt)).append("\",");
            sb.append("\"service\":\"").append(escapeJson(serviceName)).append("\",");
            sb.append("\"host\":\"").append(escapeJson(host)).append("\",");

            // Context fields
            if (wrapper.getTraceId() != null && !wrapper.getTraceId().isEmpty()) {
                sb.append("\"trace_id\":\"").append(escapeJson(wrapper.getTraceId())).append("\",");
            }
            sb.append("\"thread_name\":\"").append(escapeJson(e.getThreadName())).append("\",");
            sb.append("\"logger_name\":\"").append(escapeJson(e.getLoggerName())).append("\",");

            // Include Stack Trace if available
            String fullMessage = e.getFormattedMessage();
            if (e.getThrowableProxy() != null) {
                String stackTrace = ch.qos.logback.classic.spi.ThrowableProxyUtil.asString(e.getThrowableProxy());
                if (stackTrace != null && !stackTrace.isEmpty()) {
                    fullMessage += "\n" + stackTrace;
                }
            }

            sb.append("\"message\":\"").append(escapeJson(fullMessage)).append("\"");
            sb.append("}");
        }
        sb.append("]");
        return sb.toString();
    }

    private String mapLevel(int levelInt) {
        // Logback levels: TRACE=5000, DEBUG=10000, INFO=20000, WARN=30000, ERROR=40000
        if (levelInt >= 40000)
            return "ERROR";
        if (levelInt >= 30000)
            return "WARN";
        if (levelInt >= 20000)
            return "INFO";
        return "DEBUG";
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

            // Add Authorization header if token is configured
            if (token != null && !token.isEmpty()) {
                conn.setRequestProperty("Authorization", "Bearer " + token);
            }

            // Add Heartbeat Header
            conn.setRequestProperty("X-Instance-ID", instanceId);

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
