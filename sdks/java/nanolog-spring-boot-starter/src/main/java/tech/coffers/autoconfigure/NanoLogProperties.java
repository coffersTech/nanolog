package tech.coffers.autoconfigure;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Configuration properties for NanoLog.
 */
@Getter
@Setter
@ConfigurationProperties(prefix = "nanolog")
public class NanoLogProperties {

    /**
     * Enable NanoLog appender.
     */
    private boolean enabled = true;

    /**
     * NanoLog server URL.
     */
    private String serverUrl = "http://localhost:8080";

    /**
     * Service name to identify logs.
     */
    private String service = "default";

    /**
     * Number of log events to batch before sending.
     */
    private int batchSize = 100;

    /**
     * Maximum interval in milliseconds between flushes.
     */
    private long flushIntervalMs = 1000;

    /**
     * Enable local fallback when server is unavailable.
     * When enabled, failed logs will be written to a local file
     * and recovered when the server becomes available.
     */
    private boolean enableFallback = true;

    /**
     * Path to the fallback directory for storing failed logs.
     */
    private String fallbackPath = "/tmp/nanolog/fallback";

    /**
     * API Token for authentication with NanoLog server.
     * This token is sent in the Authorization header as Bearer token.
     */
    private String token = "";
}
