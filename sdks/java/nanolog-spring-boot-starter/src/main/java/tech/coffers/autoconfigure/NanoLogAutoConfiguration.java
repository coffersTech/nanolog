package tech.coffers.autoconfigure;

import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.LoggerContext;
import tech.coffers.logback.NanoLogAppender;
import javax.annotation.PostConstruct;
import lombok.extern.slf4j.Slf4j;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnClass;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Configuration;

/**
 * Auto-configuration for NanoLog.
 * Automatically registers NanoLogAppender to ROOT logger when enabled.
 */
@Slf4j
@Configuration
@ConditionalOnClass(NanoLogAppender.class)
@ConditionalOnProperty(prefix = "nanolog", name = "enabled", havingValue = "true", matchIfMissing = true)
@EnableConfigurationProperties(NanoLogProperties.class)
public class NanoLogAutoConfiguration {

    private final NanoLogProperties properties;

    @Value("${spring.application.name:default}")
    private String applicationName;

    public NanoLogAutoConfiguration(NanoLogProperties properties) {
        this.properties = properties;
    }

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
        appender.setServerUrl(properties.getServerUrl());

        // Use configured service name, fallback to spring.application.name
        String serviceName = "default".equals(properties.getService())
                ? applicationName
                : properties.getService();
        appender.setServiceName(serviceName);

        appender.setBatchSize(properties.getBatchSize());
        appender.setFlushIntervalMs(properties.getFlushIntervalMs());

        // Fallback configuration
        appender.setEnableFallback(properties.isEnableFallback());
        appender.setFallbackPath(properties.getFallbackPath());

        // Authentication
        appender.setToken(properties.getToken());

        appender.start();

        rootLogger.addAppender(appender);
        log.info("NanoLog auto-configured. Server: {}, Service: {}, Fallback: {}, Auth: {}",
                properties.getServerUrl(),
                serviceName,
                properties.isEnableFallback() ? properties.getFallbackPath() : "disabled",
                (properties.getToken() != null && !properties.getToken().isEmpty()) ? "enabled" : "disabled");
    }
}
