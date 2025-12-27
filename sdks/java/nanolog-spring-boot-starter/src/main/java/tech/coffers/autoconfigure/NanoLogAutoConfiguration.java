package tech.coffers.autoconfigure;

import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.LoggerContext;
import tech.coffers.logback.NanoLogAppender;
import jakarta.annotation.PostConstruct;
import lombok.extern.slf4j.Slf4j;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnClass;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Configuration;

/**
 * Auto-configuration for NanoLog.
 * Automatically registers NanoLogAppender to ROOT logger when enabled.
 */
@Slf4j
@Configuration
@ConditionalOnClass(NanoLogAppender.class)
@ConditionalOnProperty(prefix = "nanolog", name = "enabled", havingValue = "true", matchIfMissing = true)
public class NanoLogAutoConfiguration {

    @Value("${nanolog.server-url:http://localhost:8080}")
    private String serverUrl;

    @Value("${nanolog.service:${spring.application.name:default}}")
    private String serviceName;

    @Value("${nanolog.batch-size:100}")
    private int batchSize;

    @Value("${nanolog.flush-interval-ms:1000}")
    private long flushIntervalMs;

    @PostConstruct
    public void registerAppender() {
        LoggerContext context = (LoggerContext) LoggerFactory.getILoggerFactory();
        Logger rootLogger = context.getLogger(Logger.ROOT_LOGGER_NAME);

        // Check if already registered
        if (rootLogger.getAppender("NANOLOG") != null) {
            log.debug("NanoLogAppender already registered, skipping.");
            return;
        }

        NanoLogAppender appender = new NanoLogAppender();
        appender.setName("NANOLOG");
        appender.setContext(context);
        appender.setServerUrl(serverUrl);
        appender.setServiceName(serviceName);
        appender.setBatchSize(batchSize);
        appender.setFlushIntervalMs(flushIntervalMs);
        appender.start();

        rootLogger.addAppender(appender);
        log.info("NanoLog auto-configured. Server: {}, Service: {}", serverUrl, serviceName);
    }
}
